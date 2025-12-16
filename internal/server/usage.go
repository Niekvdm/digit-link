package server

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/niekvdm/digit-link/internal/db"
)

// UsageCache provides in-memory caching of usage metrics for fast quota checks
type UsageCache struct {
	db     *db.DB
	mu     sync.RWMutex
	orgs   map[string]*OrgUsage
	plans  map[string]*db.Plan // Cached plans
	planMu sync.RWMutex

	// Control channels
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// OrgUsage holds current period usage for an organization
type OrgUsage struct {
	mu sync.RWMutex

	// Cached plan ID for this org (avoids DB lookup on every request)
	planID *string

	// Baseline from DB (set on init, updated after each flush)
	dbBandwidthBytes int64
	dbTunnelSeconds  int64
	dbRequestCount   int64

	// Delta since last flush (reset to 0 after flush)
	deltaBandwidthBytes int64
	deltaTunnelSeconds  int64
	deltaRequestCount   int64

	ConcurrentTunnels int32 // atomic
	PeriodStart       time.Time
	LimitHitAt        *time.Time
	lastFlush         time.Time
}

// NewUsageCache creates a new usage cache
func NewUsageCache(database *db.DB) *UsageCache {
	uc := &UsageCache{
		db:     database,
		orgs:   make(map[string]*OrgUsage),
		plans:  make(map[string]*db.Plan),
		stopCh: make(chan struct{}),
	}

	// Load plans into cache
	uc.refreshPlansCache()

	return uc
}

// Start begins the background sync goroutine
func (uc *UsageCache) Start() {
	uc.wg.Add(1)
	go uc.syncLoop()
}

// Stop stops the background sync goroutine
func (uc *UsageCache) Stop() {
	close(uc.stopCh)
	uc.wg.Wait()
	// Final flush
	uc.flushAll()
}

// syncLoop periodically syncs usage to the database
func (uc *UsageCache) syncLoop() {
	defer uc.wg.Done()

	flushTicker := time.NewTicker(1 * time.Minute)
	plansTicker := time.NewTicker(5 * time.Minute)
	rollupTicker := time.NewTicker(1 * time.Hour)
	defer flushTicker.Stop()
	defer plansTicker.Stop()
	defer rollupTicker.Stop()

	for {
		select {
		case <-uc.stopCh:
			return
		case <-flushTicker.C:
			uc.flushAll()
		case <-plansTicker.C:
			uc.refreshPlansCache()
		case <-rollupTicker.C:
			uc.runRollups()
		}
	}
}

// flushAll syncs all cached usage to the database
func (uc *UsageCache) flushAll() {
	uc.mu.RLock()
	orgIDs := make([]string, 0, len(uc.orgs))
	for orgID := range uc.orgs {
		orgIDs = append(orgIDs, orgID)
	}
	uc.mu.RUnlock()

	for _, orgID := range orgIDs {
		uc.flushOrg(orgID)
	}
}

// flushOrg syncs a single org's usage to the database
func (uc *UsageCache) flushOrg(orgID string) {
	uc.mu.RLock()
	usage, exists := uc.orgs[orgID]
	uc.mu.RUnlock()

	if !exists {
		return
	}

	usage.mu.Lock()
	defer usage.mu.Unlock()

	// Only flush if there's delta data to flush
	if usage.deltaBandwidthBytes == 0 && usage.deltaTunnelSeconds == 0 && usage.deltaRequestCount == 0 {
		return
	}

	now := time.Now()
	hourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)

	err := uc.db.UpsertUsageSnapshot(
		orgID,
		db.PeriodHourly,
		hourStart,
		usage.deltaBandwidthBytes,
		usage.deltaTunnelSeconds,
		usage.deltaRequestCount,
		int(atomic.LoadInt32(&usage.ConcurrentTunnels)),
	)
	if err != nil {
		log.Printf("Failed to flush usage for org %s: %v", orgID, err)
		return
	}

	// Move delta to baseline and reset delta counters
	usage.dbBandwidthBytes += usage.deltaBandwidthBytes
	usage.dbTunnelSeconds += usage.deltaTunnelSeconds
	usage.dbRequestCount += usage.deltaRequestCount

	usage.deltaBandwidthBytes = 0
	usage.deltaTunnelSeconds = 0
	usage.deltaRequestCount = 0
	usage.lastFlush = now
}

// refreshPlansCache reloads plans from the database
func (uc *UsageCache) refreshPlansCache() {
	plans, err := uc.db.ListPlans()
	if err != nil {
		log.Printf("Failed to refresh plans cache: %v", err)
		return
	}

	uc.planMu.Lock()
	defer uc.planMu.Unlock()

	uc.plans = make(map[string]*db.Plan)
	for _, p := range plans {
		uc.plans[p.ID] = p
	}
}

// GetPlan returns a cached plan
func (uc *UsageCache) GetPlan(planID string) *db.Plan {
	uc.planMu.RLock()
	defer uc.planMu.RUnlock()
	return uc.plans[planID]
}

// getOrCreateOrgUsage gets or creates usage tracking for an org
func (uc *UsageCache) getOrCreateOrgUsage(orgID string) *OrgUsage {
	uc.mu.RLock()
	usage, exists := uc.orgs[orgID]
	uc.mu.RUnlock()

	if exists {
		return usage
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	// Double-check after acquiring write lock
	if usage, exists = uc.orgs[orgID]; exists {
		return usage
	}

	// Get current period start
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Load existing usage from database
	existing, err := uc.db.GetCurrentPeriodUsage(orgID)
	if err != nil {
		log.Printf("Failed to load existing usage for org %s: %v", orgID, err)
	}

	// Load organization to get planID (cached to avoid DB query on every request)
	var planID *string
	org, err := uc.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Printf("Failed to load organization for org %s: %v", orgID, err)
	} else if org != nil {
		planID = org.PlanID
	}

	usage = &OrgUsage{
		PeriodStart: periodStart,
		lastFlush:   now,
		planID:      planID,
	}

	// Load DB totals into baseline fields (delta starts at 0)
	if existing != nil {
		usage.dbBandwidthBytes = existing.BandwidthBytes
		usage.dbTunnelSeconds = existing.TunnelSeconds
		usage.dbRequestCount = existing.RequestCount
	}

	uc.orgs[orgID] = usage
	return usage
}

// RecordBandwidth records bandwidth usage for an organization
func (uc *UsageCache) RecordBandwidth(orgID string, bytes int64) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.Lock()
	usage.deltaBandwidthBytes += bytes
	usage.mu.Unlock()
}

// RecordRequest records a request for an organization
func (uc *UsageCache) RecordRequest(orgID string) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.Lock()
	usage.deltaRequestCount++
	usage.mu.Unlock()
}

// RecordTunnelTime records tunnel duration for an organization
func (uc *UsageCache) RecordTunnelTime(orgID string, seconds int64) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.Lock()
	usage.deltaTunnelSeconds += seconds
	usage.mu.Unlock()
}

// IncrementConcurrentTunnels increments concurrent tunnel count and updates peak
func (uc *UsageCache) IncrementConcurrentTunnels(orgID string) int32 {
	usage := uc.getOrCreateOrgUsage(orgID)
	return atomic.AddInt32(&usage.ConcurrentTunnels, 1)
}

// DecrementConcurrentTunnels decrements concurrent tunnel count
func (uc *UsageCache) DecrementConcurrentTunnels(orgID string) int32 {
	usage := uc.getOrCreateOrgUsage(orgID)
	return atomic.AddInt32(&usage.ConcurrentTunnels, -1)
}

// GetConcurrentTunnels returns the current concurrent tunnel count
func (uc *UsageCache) GetConcurrentTunnels(orgID string) int32 {
	usage := uc.getOrCreateOrgUsage(orgID)
	return atomic.LoadInt32(&usage.ConcurrentTunnels)
}

// GetOrgPlanID returns the cached plan ID for an organization
func (uc *UsageCache) GetOrgPlanID(orgID string) *string {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.RLock()
	defer usage.mu.RUnlock()
	return usage.planID
}

// GetCurrentUsage returns current usage for an organization (db baseline + unflushed delta)
func (uc *UsageCache) GetCurrentUsage(orgID string) (bandwidth, tunnelSeconds, requests int64, concurrent int32) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.RLock()
	bandwidth = usage.dbBandwidthBytes + usage.deltaBandwidthBytes
	tunnelSeconds = usage.dbTunnelSeconds + usage.deltaTunnelSeconds
	requests = usage.dbRequestCount + usage.deltaRequestCount
	usage.mu.RUnlock()
	concurrent = atomic.LoadInt32(&usage.ConcurrentTunnels)
	return
}

// SetLimitHit marks when a limit was hit
func (uc *UsageCache) SetLimitHit(orgID string) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.Lock()
	if usage.LimitHitAt == nil {
		now := time.Now()
		usage.LimitHitAt = &now
	}
	usage.mu.Unlock()
}

// GetLimitHitTime returns when the limit was hit (nil if not hit)
func (uc *UsageCache) GetLimitHitTime(orgID string) *time.Time {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.RLock()
	defer usage.mu.RUnlock()
	return usage.LimitHitAt
}

// ClearLimitHit clears the limit hit marker (e.g., after upgrade or new period)
func (uc *UsageCache) ClearLimitHit(orgID string) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.Lock()
	usage.LimitHitAt = nil
	usage.mu.Unlock()
}

// ResetOrgUsage resets usage counters for an organization (admin action)
func (uc *UsageCache) ResetOrgUsage(orgID string) {
	usage := uc.getOrCreateOrgUsage(orgID)
	usage.mu.Lock()
	usage.dbBandwidthBytes = 0
	usage.dbTunnelSeconds = 0
	usage.dbRequestCount = 0
	usage.deltaBandwidthBytes = 0
	usage.deltaTunnelSeconds = 0
	usage.deltaRequestCount = 0
	usage.LimitHitAt = nil
	usage.mu.Unlock()
}

// Retention periods
const (
	HourlyRetention = 7 * 24 * time.Hour  // 7 days
	DailyRetention  = 90 * 24 * time.Hour // 90 days
)

// runRollups performs periodic rollup and cleanup operations
func (uc *UsageCache) runRollups() {
	now := time.Now()

	// Daily rollup: aggregate yesterday's hourly data into a daily snapshot
	// Run at the start of each day (when hour is 0-1)
	if now.Hour() < 1 {
		yesterday := now.AddDate(0, 0, -1)
		if err := uc.db.RollupHourlyToDaily(yesterday); err != nil {
			log.Printf("Failed to rollup hourly to daily: %v", err)
		} else {
			log.Printf("Completed daily rollup for %s", yesterday.Format("2006-01-02"))
		}
	}

	// Monthly rollup: aggregate last month's daily data into a monthly snapshot
	// Run on the 1st of each month
	if now.Day() == 1 && now.Hour() < 1 {
		lastMonth := now.AddDate(0, -1, 0)
		if err := uc.db.RollupDailyToMonthly(lastMonth); err != nil {
			log.Printf("Failed to rollup daily to monthly: %v", err)
		} else {
			log.Printf("Completed monthly rollup for %s", lastMonth.Format("2006-01"))
		}
	}

	// Cleanup old hourly snapshots (older than 7 days)
	if deleted, err := uc.db.CleanupOldHourlySnapshots(HourlyRetention); err != nil {
		log.Printf("Failed to cleanup old hourly snapshots: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d old hourly snapshots", deleted)
	}

	// Cleanup old daily snapshots (older than 90 days)
	if deleted, err := uc.db.CleanupOldDailySnapshots(DailyRetention); err != nil {
		log.Printf("Failed to cleanup old daily snapshots: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d old daily snapshots", deleted)
	}
}
