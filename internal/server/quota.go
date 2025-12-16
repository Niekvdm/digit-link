package server

import (
	"time"

	"github.com/niekvdm/digit-link/internal/db"
)

// QuotaType represents the type of quota being checked
type QuotaType int

const (
	QuotaBandwidth QuotaType = iota
	QuotaTunnelHours
	QuotaConcurrentTunnels
	QuotaRequests
)

// QuotaResult represents the result of a quota check
type QuotaResult struct {
	Allowed       bool
	Remaining     int64
	Limit         int64
	Used          int64
	ResetTime     time.Time
	InGracePeriod bool
}

// QuotaChecker handles quota enforcement
type QuotaChecker struct {
	cache *UsageCache
	db    *db.DB
}

// NewQuotaChecker creates a new quota checker
func NewQuotaChecker(cache *UsageCache, database *db.DB) *QuotaChecker {
	return &QuotaChecker{
		cache: cache,
		db:    database,
	}
}

// CheckQuota checks if a specific quota is within limits for an organization
func (qc *QuotaChecker) CheckQuota(orgID string, quotaType QuotaType) QuotaResult {
	result := QuotaResult{
		Allowed:   true,
		Remaining: -1, // Unlimited
		Limit:     -1,
	}

	// Get organization and plan
	org, err := qc.db.GetOrganizationByID(orgID)
	if err != nil || org == nil || org.PlanID == nil {
		// No plan = no limits
		return result
	}

	plan := qc.cache.GetPlan(*org.PlanID)
	if plan == nil {
		// Plan not found in cache, try to reload
		plan, err = qc.db.GetPlan(*org.PlanID)
		if err != nil || plan == nil {
			return result
		}
	}

	// Get current usage
	bandwidth, tunnelSeconds, requests, concurrent := qc.cache.GetCurrentUsage(orgID)

	// Calculate period reset time
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	result.ResetTime = periodStart.AddDate(0, 1, 0)

	// Check the specific quota type
	var limit *int64
	var used int64

	switch quotaType {
	case QuotaBandwidth:
		if plan.BandwidthBytesMonthly != nil {
			limit = plan.BandwidthBytesMonthly
			used = bandwidth
		}
	case QuotaTunnelHours:
		if plan.TunnelHoursMonthly != nil {
			// Convert seconds to hours for comparison
			limitHours := *plan.TunnelHoursMonthly
			limitSeconds := limitHours * 3600
			limit = &limitSeconds
			used = tunnelSeconds
		}
	case QuotaConcurrentTunnels:
		if plan.ConcurrentTunnelsMax != nil {
			limitInt := int64(*plan.ConcurrentTunnelsMax)
			limit = &limitInt
			used = int64(concurrent)
		}
	case QuotaRequests:
		if plan.RequestsMonthly != nil {
			limit = plan.RequestsMonthly
			used = requests
		}
	}

	if limit == nil {
		// No limit set for this quota type
		return result
	}

	result.Limit = *limit
	result.Used = used

	// Calculate effective limit with overage allowance
	effectiveLimit := *limit
	if plan.OverageAllowedPercent > 0 {
		effectiveLimit = *limit * int64(100+plan.OverageAllowedPercent) / 100
	}

	result.Remaining = effectiveLimit - used
	if result.Remaining < 0 {
		result.Remaining = 0
	}

	if used >= effectiveLimit {
		// Check grace period
		if plan.GracePeriodHours > 0 {
			limitHitAt := qc.cache.GetLimitHitTime(orgID)
			if limitHitAt == nil {
				// First time hitting limit
				qc.cache.SetLimitHit(orgID)
				result.Allowed = true
				result.InGracePeriod = true
				return result
			}

			graceDuration := time.Duration(plan.GracePeriodHours) * time.Hour
			if time.Since(*limitHitAt) < graceDuration {
				// Still in grace period
				result.Allowed = true
				result.InGracePeriod = true
				return result
			}
		}

		// Not allowed
		result.Allowed = false
		return result
	}

	// Within limits
	return result
}

// CheckAllQuotas checks all quotas for an organization
func (qc *QuotaChecker) CheckAllQuotas(orgID string) map[QuotaType]QuotaResult {
	results := make(map[QuotaType]QuotaResult)
	results[QuotaBandwidth] = qc.CheckQuota(orgID, QuotaBandwidth)
	results[QuotaTunnelHours] = qc.CheckQuota(orgID, QuotaTunnelHours)
	results[QuotaConcurrentTunnels] = qc.CheckQuota(orgID, QuotaConcurrentTunnels)
	results[QuotaRequests] = qc.CheckQuota(orgID, QuotaRequests)
	return results
}

// CanConnectTunnel checks if a new tunnel connection is allowed
func (qc *QuotaChecker) CanConnectTunnel(orgID string) (allowed bool, reason string) {
	// Check concurrent tunnels limit
	result := qc.CheckQuota(orgID, QuotaConcurrentTunnels)
	if !result.Allowed {
		return false, "concurrent tunnel limit exceeded"
	}

	// Soft check bandwidth - warn if near limit but allow connection
	bandwidthResult := qc.CheckQuota(orgID, QuotaBandwidth)
	if !bandwidthResult.Allowed && !bandwidthResult.InGracePeriod {
		return false, "monthly bandwidth limit exceeded"
	}

	return true, ""
}

// CanProcessRequest checks if a request can be processed
func (qc *QuotaChecker) CanProcessRequest(orgID string) (allowed bool, reason string) {
	// Check request limit
	result := qc.CheckQuota(orgID, QuotaRequests)
	if !result.Allowed && !result.InGracePeriod {
		return false, "monthly request limit exceeded"
	}

	// Check bandwidth (will be checked again after request completes)
	bandwidthResult := qc.CheckQuota(orgID, QuotaBandwidth)
	if !bandwidthResult.Allowed && !bandwidthResult.InGracePeriod {
		return false, "monthly bandwidth limit exceeded"
	}

	return true, ""
}

// GetQuotaHeaders returns HTTP headers for quota information
func (qc *QuotaChecker) GetQuotaHeaders(orgID string, quotaType QuotaType) map[string]string {
	result := qc.CheckQuota(orgID, quotaType)
	headers := make(map[string]string)

	if result.Limit > 0 {
		headers["X-Quota-Limit"] = formatInt64(result.Limit)
		headers["X-Quota-Used"] = formatInt64(result.Used)
		headers["X-Quota-Remaining"] = formatInt64(result.Remaining)
		headers["X-Quota-Reset"] = result.ResetTime.Format(time.RFC3339)
	}

	return headers
}

func formatInt64(n int64) string {
	// Simple int64 to string conversion
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte(n%10) + '0'
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
