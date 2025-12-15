package policy

import (
	"fmt"
	"sync"
	"time"

	"github.com/niekvdm/digit-link/internal/db"
)

// Loader provides cached policy loading with automatic refresh
type Loader struct {
	db       *db.DB
	resolver *Resolver

	// Cache for policies
	orgPolicies map[string]*cachedPolicy
	appPolicies map[string]*cachedPolicy
	mu          sync.RWMutex

	// Cache configuration
	cacheTTL time.Duration
}

type cachedPolicy struct {
	policy    *EffectivePolicy
	loadedAt  time.Time
	expiresAt time.Time
}

// LoaderOption is a functional option for configuring the loader
type LoaderOption func(*Loader)

// WithCacheTTL sets the cache TTL for policies
func WithCacheTTL(ttl time.Duration) LoaderOption {
	return func(l *Loader) {
		l.cacheTTL = ttl
	}
}

// NewLoader creates a new policy loader
func NewLoader(database *db.DB, resolver *Resolver, opts ...LoaderOption) *Loader {
	l := &Loader{
		db:          database,
		resolver:    resolver,
		orgPolicies: make(map[string]*cachedPolicy),
		appPolicies: make(map[string]*cachedPolicy),
		cacheTTL:    5 * time.Minute, // Default 5 minute cache
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoadForSubdomain loads the effective policy for a subdomain
func (l *Loader) LoadForSubdomain(subdomain string) (*EffectivePolicy, *AuthContext, error) {
	return l.resolver.ResolveForSubdomain(subdomain)
}

// LoadForOrg loads the effective policy for an organization (with caching)
func (l *Loader) LoadForOrg(orgID string) (*EffectivePolicy, error) {
	// Check cache first
	l.mu.RLock()
	cached, ok := l.orgPolicies[orgID]
	l.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached.policy, nil
	}

	// Load from database
	policy, err := l.resolver.resolveForOrg(orgID)
	if err != nil {
		return nil, err
	}

	// Update cache
	l.mu.Lock()
	l.orgPolicies[orgID] = &cachedPolicy{
		policy:    policy,
		loadedAt:  time.Now(),
		expiresAt: time.Now().Add(l.cacheTTL),
	}
	l.mu.Unlock()

	return policy, nil
}

// LoadForApp loads the effective policy for an application (with caching)
func (l *Loader) LoadForApp(appID string) (*EffectivePolicy, error) {
	// Check cache first
	l.mu.RLock()
	cached, ok := l.appPolicies[appID]
	l.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached.policy, nil
	}

	// Load app from database
	app, err := l.db.GetApplicationByID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to load application: %w", err)
	}
	if app == nil {
		return nil, nil
	}

	ctx := &AuthContext{
		Subdomain:       app.Subdomain,
		AppID:           app.ID,
		OrgID:           app.OrgID,
		App:             app,
		IsPersistentApp: true,
	}

	policy, err := l.resolver.ResolveForContext(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	l.mu.Lock()
	l.appPolicies[appID] = &cachedPolicy{
		policy:    policy,
		loadedAt:  time.Now(),
		expiresAt: time.Now().Add(l.cacheTTL),
	}
	l.mu.Unlock()

	return policy, nil
}

// InvalidateOrg removes an organization's policy from cache
func (l *Loader) InvalidateOrg(orgID string) {
	l.mu.Lock()
	delete(l.orgPolicies, orgID)
	l.mu.Unlock()
}

// InvalidateApp removes an application's policy from cache
func (l *Loader) InvalidateApp(appID string) {
	l.mu.Lock()
	delete(l.appPolicies, appID)
	l.mu.Unlock()
}

// InvalidateAll clears all cached policies
func (l *Loader) InvalidateAll() {
	l.mu.Lock()
	l.orgPolicies = make(map[string]*cachedPolicy)
	l.appPolicies = make(map[string]*cachedPolicy)
	l.mu.Unlock()
}

// CacheStats returns cache statistics
type CacheStats struct {
	OrgPoliciesCached int `json:"orgPoliciesCached"`
	AppPoliciesCached int `json:"appPoliciesCached"`
}

func (l *Loader) CacheStats() CacheStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return CacheStats{
		OrgPoliciesCached: len(l.orgPolicies),
		AppPoliciesCached: len(l.appPolicies),
	}
}

// StartCleanup starts a background goroutine to clean up expired cache entries
func (l *Loader) StartCleanup(interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				l.cleanupExpired()
			case <-stop:
				return
			}
		}
	}()
	return stop
}

func (l *Loader) cleanupExpired() {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for orgID, cached := range l.orgPolicies {
		if now.After(cached.expiresAt) {
			delete(l.orgPolicies, orgID)
		}
	}

	for appID, cached := range l.appPolicies {
		if now.After(cached.expiresAt) {
			delete(l.appPolicies, appID)
		}
	}
}
