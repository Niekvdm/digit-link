package auth

import (
	"database/sql"
	"sync"
	"time"

	"github.com/niekvdm/digit-link/internal/db"
)

// RateLimiter provides SQLite-backed rate limiting for auth endpoints
type RateLimiter struct {
	db *db.DB

	// Configuration
	windowDuration  time.Duration
	maxAttempts     int
	blockDuration   time.Duration
	cleanupInterval time.Duration

	// In-memory cache for hot paths (backed by SQLite)
	cache   map[string]*rateLimitEntry
	cacheMu sync.RWMutex

	// Stop channel for cleanup goroutine
	stopCleanup chan struct{}
}

type rateLimitEntry struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
}

// RateLimiterConfig holds configuration for the rate limiter
type RateLimiterConfig struct {
	// WindowDuration is the time window for counting attempts
	WindowDuration time.Duration
	// MaxAttempts is the maximum number of attempts allowed in the window
	MaxAttempts int
	// BlockDuration is how long to block after exceeding max attempts
	BlockDuration time.Duration
	// CleanupInterval is how often to clean up expired entries
	CleanupInterval time.Duration
}

// DefaultRateLimiterConfig returns the default rate limiter configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		WindowDuration:  15 * time.Minute,
		MaxAttempts:     10,
		BlockDuration:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(database *db.DB, config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		db:              database,
		windowDuration:  config.WindowDuration,
		maxAttempts:     config.MaxAttempts,
		blockDuration:   config.BlockDuration,
		cleanupInterval: config.CleanupInterval,
		cache:           make(map[string]*rateLimitEntry),
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request should be allowed based on rate limits
// Returns (allowed, retryAfter) where retryAfter is the duration until the block expires
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.cacheMu.Lock()
	defer rl.cacheMu.Unlock()

	now := time.Now()

	// Check cache first
	entry, ok := rl.cache[key]
	if !ok {
		// Load from database or create new
		entry = rl.loadFromDB(key)
		if entry == nil {
			entry = &rateLimitEntry{
				count:       0,
				windowStart: now,
			}
		}
		rl.cache[key] = entry
	}

	// Check if blocked
	if !entry.blockedUntil.IsZero() && now.Before(entry.blockedUntil) {
		return false, entry.blockedUntil.Sub(now)
	}

	// Check if window has expired
	if now.Sub(entry.windowStart) > rl.windowDuration {
		// Reset window
		entry.count = 0
		entry.windowStart = now
		entry.blockedUntil = time.Time{}
	}

	// Increment count
	entry.count++

	// Check if over limit
	if entry.count > rl.maxAttempts {
		entry.blockedUntil = now.Add(rl.blockDuration)
		rl.saveToDB(key, entry)
		return false, rl.blockDuration
	}

	// Save to DB periodically (every 5 attempts or when close to limit)
	if entry.count%5 == 0 || entry.count >= rl.maxAttempts-2 {
		rl.saveToDB(key, entry)
	}

	return true, 0
}

// RecordSuccess records a successful auth attempt (can reset rate limit)
func (rl *RateLimiter) RecordSuccess(key string) {
	rl.cacheMu.Lock()
	defer rl.cacheMu.Unlock()

	// On success, we can optionally reset the counter
	// For security, we keep some record but reset the block
	if entry, ok := rl.cache[key]; ok {
		entry.blockedUntil = time.Time{}
		// Don't fully reset count to prevent timing attacks
		if entry.count > 0 {
			entry.count = entry.count / 2
		}
		rl.saveToDB(key, entry)
	}
}

// RecordFailure records a failed auth attempt
func (rl *RateLimiter) RecordFailure(key string) {
	// Allow already increments the counter, but we can use this
	// to explicitly record a failure and potentially trigger immediate save
	rl.cacheMu.Lock()
	defer rl.cacheMu.Unlock()

	if entry, ok := rl.cache[key]; ok {
		rl.saveToDB(key, entry)
	}
}

// IsBlocked checks if a key is currently blocked
func (rl *RateLimiter) IsBlocked(key string) (bool, time.Duration) {
	rl.cacheMu.RLock()
	entry, ok := rl.cache[key]
	rl.cacheMu.RUnlock()

	if !ok {
		// Check database
		entry = rl.loadFromDB(key)
		if entry == nil {
			return false, 0
		}
	}

	now := time.Now()
	if !entry.blockedUntil.IsZero() && now.Before(entry.blockedUntil) {
		return true, entry.blockedUntil.Sub(now)
	}

	return false, 0
}

// Reset clears the rate limit for a key (admin use)
func (rl *RateLimiter) Reset(key string) {
	rl.cacheMu.Lock()
	delete(rl.cache, key)
	rl.cacheMu.Unlock()

	// Delete from database
	rl.db.Conn().Exec("DELETE FROM rate_limit_state WHERE key = ?", key)
}

// GetStats returns rate limiting statistics for a key
func (rl *RateLimiter) GetStats(key string) (count int, windowStart time.Time, blockedUntil time.Time) {
	rl.cacheMu.RLock()
	entry, ok := rl.cache[key]
	rl.cacheMu.RUnlock()

	if !ok {
		entry = rl.loadFromDB(key)
		if entry == nil {
			return 0, time.Time{}, time.Time{}
		}
	}

	return entry.count, entry.windowStart, entry.blockedUntil
}

// loadFromDB loads a rate limit entry from the database
func (rl *RateLimiter) loadFromDB(key string) *rateLimitEntry {
	if rl.db == nil {
		return nil
	}

	var count int
	var windowStart, blockedUntil sql.NullTime

	err := rl.db.Conn().QueryRow(`
		SELECT count, window_start, blocked_until
		FROM rate_limit_state WHERE key = ?
	`, key).Scan(&count, &windowStart, &blockedUntil)

	if err != nil {
		return nil
	}

	entry := &rateLimitEntry{
		count: count,
	}
	if windowStart.Valid {
		entry.windowStart = windowStart.Time
	}
	if blockedUntil.Valid {
		entry.blockedUntil = blockedUntil.Time
	}

	return entry
}

// saveToDB saves a rate limit entry to the database
func (rl *RateLimiter) saveToDB(key string, entry *rateLimitEntry) {
	if rl.db == nil {
		return
	}

	var blockedUntil interface{}
	if !entry.blockedUntil.IsZero() {
		blockedUntil = entry.blockedUntil
	}

	rl.db.Conn().Exec(`
		INSERT INTO rate_limit_state (key, count, window_start, blocked_until)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			count = excluded.count,
			window_start = excluded.window_start,
			blocked_until = excluded.blocked_until
	`, key, entry.count, entry.windowStart, blockedUntil)
}

// cleanupLoop periodically cleans up expired entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries from cache and database
func (rl *RateLimiter) cleanup() {
	now := time.Now()

	// Clean cache
	rl.cacheMu.Lock()
	for key, entry := range rl.cache {
		// Remove if window expired and not blocked
		if now.Sub(entry.windowStart) > rl.windowDuration &&
			(entry.blockedUntil.IsZero() || now.After(entry.blockedUntil)) {
			delete(rl.cache, key)
		}
	}
	rl.cacheMu.Unlock()

	// Clean database
	if rl.db != nil {
		cutoff := now.Add(-rl.windowDuration)
		rl.db.Conn().Exec(`
			DELETE FROM rate_limit_state 
			WHERE window_start < ? 
			AND (blocked_until IS NULL OR blocked_until < ?)
		`, cutoff, now)
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// BuildKey builds a rate limit key from components
func BuildRateLimitKey(keyType string, components ...string) string {
	key := keyType
	for _, c := range components {
		key += ":" + c
	}
	return key
}

// Common key builders

// IPRateLimitKey returns a rate limit key for an IP address
func IPRateLimitKey(ip string) string {
	return BuildRateLimitKey("ip", ip)
}

// AppIPRateLimitKey returns a rate limit key for an IP + app combination
func AppIPRateLimitKey(appID, ip string) string {
	return BuildRateLimitKey("app_ip", appID, ip)
}

// OrgIPRateLimitKey returns a rate limit key for an IP + org combination
func OrgIPRateLimitKey(orgID, ip string) string {
	return BuildRateLimitKey("org_ip", orgID, ip)
}

// UserRateLimitKey returns a rate limit key for a user identity
func UserRateLimitKey(userIdentity string) string {
	return BuildRateLimitKey("user", userIdentity)
}
