package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PeriodType represents the type of usage snapshot period
type PeriodType string

const (
	PeriodHourly  PeriodType = "hourly"
	PeriodDaily   PeriodType = "daily"
	PeriodMonthly PeriodType = "monthly"
)

// UsageSnapshot represents aggregated usage metrics for a period
type UsageSnapshot struct {
	ID                    string     `json:"id"`
	OrgID                 string     `json:"orgId"`
	PeriodType            PeriodType `json:"periodType"`
	PeriodStart           time.Time  `json:"periodStart"`
	BandwidthBytes        int64      `json:"bandwidthBytes"`
	TunnelSeconds         int64      `json:"tunnelSeconds"`
	RequestCount          int64      `json:"requestCount"`
	PeakConcurrentTunnels int        `json:"peakConcurrentTunnels"`
}

// CreateUsageSnapshot creates a new usage snapshot
func (db *DB) CreateUsageSnapshot(orgID string, periodType PeriodType, periodStart time.Time) (*UsageSnapshot, error) {
	id := uuid.New().String()

	_, err := db.conn.Exec(`
		INSERT INTO usage_snapshots (id, org_id, period_type, period_start)
		VALUES (?, ?, ?, ?)
	`, id, orgID, string(periodType), periodStart)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage snapshot: %w", err)
	}

	return &UsageSnapshot{
		ID:          id,
		OrgID:       orgID,
		PeriodType:  periodType,
		PeriodStart: periodStart,
	}, nil
}

// UpsertUsageSnapshot creates or updates a usage snapshot with the given metrics
func (db *DB) UpsertUsageSnapshot(orgID string, periodType PeriodType, periodStart time.Time,
	bandwidthBytes, tunnelSeconds, requestCount int64, peakConcurrent int) error {

	_, err := db.conn.Exec(`
		INSERT INTO usage_snapshots (id, org_id, period_type, period_start, 
			bandwidth_bytes, tunnel_seconds, request_count, peak_concurrent_tunnels)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(org_id, period_type, period_start) DO UPDATE SET
			bandwidth_bytes = bandwidth_bytes + excluded.bandwidth_bytes,
			tunnel_seconds = tunnel_seconds + excluded.tunnel_seconds,
			request_count = request_count + excluded.request_count,
			peak_concurrent_tunnels = MAX(peak_concurrent_tunnels, excluded.peak_concurrent_tunnels)
	`, uuid.New().String(), orgID, string(periodType), periodStart,
		bandwidthBytes, tunnelSeconds, requestCount, peakConcurrent)

	return err
}

// IncrementUsageSnapshot increments the counters for an existing snapshot
func (db *DB) IncrementUsageSnapshot(orgID string, periodType PeriodType, periodStart time.Time,
	bandwidthBytes, tunnelSeconds, requestCount int64, peakConcurrent int) error {

	result, err := db.conn.Exec(`
		UPDATE usage_snapshots SET
			bandwidth_bytes = bandwidth_bytes + ?,
			tunnel_seconds = tunnel_seconds + ?,
			request_count = request_count + ?,
			peak_concurrent_tunnels = MAX(peak_concurrent_tunnels, ?)
		WHERE org_id = ? AND period_type = ? AND period_start = ?
	`, bandwidthBytes, tunnelSeconds, requestCount, peakConcurrent,
		orgID, string(periodType), periodStart)
	if err != nil {
		return fmt.Errorf("failed to increment usage snapshot: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		// Create new snapshot if it doesn't exist
		return db.UpsertUsageSnapshot(orgID, periodType, periodStart,
			bandwidthBytes, tunnelSeconds, requestCount, peakConcurrent)
	}

	return nil
}

// GetUsageSnapshot retrieves a specific usage snapshot
func (db *DB) GetUsageSnapshot(orgID string, periodType PeriodType, periodStart time.Time) (*UsageSnapshot, error) {
	snapshot := &UsageSnapshot{}

	err := db.conn.QueryRow(`
		SELECT id, org_id, period_type, period_start, bandwidth_bytes,
		       tunnel_seconds, request_count, peak_concurrent_tunnels
		FROM usage_snapshots
		WHERE org_id = ? AND period_type = ? AND period_start = ?
	`, orgID, string(periodType), periodStart).Scan(
		&snapshot.ID, &snapshot.OrgID, &snapshot.PeriodType, &snapshot.PeriodStart,
		&snapshot.BandwidthBytes, &snapshot.TunnelSeconds, &snapshot.RequestCount,
		&snapshot.PeakConcurrentTunnels,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get usage snapshot: %w", err)
	}

	return snapshot, nil
}

// GetUsageSnapshotsForOrg retrieves all usage snapshots for an organization within a time range
func (db *DB) GetUsageSnapshotsForOrg(orgID string, periodType PeriodType, start, end time.Time) ([]*UsageSnapshot, error) {
	rows, err := db.conn.Query(`
		SELECT id, org_id, period_type, period_start, bandwidth_bytes,
		       tunnel_seconds, request_count, peak_concurrent_tunnels
		FROM usage_snapshots
		WHERE org_id = ? AND period_type = ? AND period_start >= ? AND period_start < ?
		ORDER BY period_start
	`, orgID, string(periodType), start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*UsageSnapshot
	for rows.Next() {
		snapshot := &UsageSnapshot{}
		err := rows.Scan(
			&snapshot.ID, &snapshot.OrgID, &snapshot.PeriodType, &snapshot.PeriodStart,
			&snapshot.BandwidthBytes, &snapshot.TunnelSeconds, &snapshot.RequestCount,
			&snapshot.PeakConcurrentTunnels,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

// GetCurrentPeriodUsage returns the aggregated usage for the current billing period (month)
func (db *DB) GetCurrentPeriodUsage(orgID string) (*UsageSnapshot, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	snapshot := &UsageSnapshot{
		OrgID:       orgID,
		PeriodType:  PeriodMonthly,
		PeriodStart: periodStart,
	}

	// Aggregate from all snapshot types for the current month
	err := db.conn.QueryRow(`
		SELECT COALESCE(SUM(bandwidth_bytes), 0), COALESCE(SUM(tunnel_seconds), 0),
		       COALESCE(SUM(request_count), 0), COALESCE(MAX(peak_concurrent_tunnels), 0)
		FROM usage_snapshots
		WHERE org_id = ? AND period_start >= ? AND period_start < ?
	`, orgID, periodStart, periodEnd).Scan(
		&snapshot.BandwidthBytes, &snapshot.TunnelSeconds,
		&snapshot.RequestCount, &snapshot.PeakConcurrentTunnels,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get current period usage: %w", err)
	}

	return snapshot, nil
}

// RollupHourlyToDaily aggregates hourly snapshots into daily snapshots
func (db *DB) RollupHourlyToDaily(date time.Time) error {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	_, err := db.conn.Exec(`
		INSERT INTO usage_snapshots (id, org_id, period_type, period_start,
			bandwidth_bytes, tunnel_seconds, request_count, peak_concurrent_tunnels)
		SELECT 
			lower(hex(randomblob(16))),
			org_id,
			'daily',
			?,
			SUM(bandwidth_bytes),
			SUM(tunnel_seconds),
			SUM(request_count),
			MAX(peak_concurrent_tunnels)
		FROM usage_snapshots
		WHERE period_type = 'hourly' AND period_start >= ? AND period_start < ?
		GROUP BY org_id
		ON CONFLICT(org_id, period_type, period_start) DO UPDATE SET
			bandwidth_bytes = excluded.bandwidth_bytes,
			tunnel_seconds = excluded.tunnel_seconds,
			request_count = excluded.request_count,
			peak_concurrent_tunnels = excluded.peak_concurrent_tunnels
	`, dayStart, dayStart, dayEnd)

	return err
}

// RollupDailyToMonthly aggregates daily snapshots into monthly snapshots
func (db *DB) RollupDailyToMonthly(month time.Time) error {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	_, err := db.conn.Exec(`
		INSERT INTO usage_snapshots (id, org_id, period_type, period_start,
			bandwidth_bytes, tunnel_seconds, request_count, peak_concurrent_tunnels)
		SELECT 
			lower(hex(randomblob(16))),
			org_id,
			'monthly',
			?,
			SUM(bandwidth_bytes),
			SUM(tunnel_seconds),
			SUM(request_count),
			MAX(peak_concurrent_tunnels)
		FROM usage_snapshots
		WHERE period_type = 'daily' AND period_start >= ? AND period_start < ?
		GROUP BY org_id
		ON CONFLICT(org_id, period_type, period_start) DO UPDATE SET
			bandwidth_bytes = excluded.bandwidth_bytes,
			tunnel_seconds = excluded.tunnel_seconds,
			request_count = excluded.request_count,
			peak_concurrent_tunnels = excluded.peak_concurrent_tunnels
	`, monthStart, monthStart, monthEnd)

	return err
}

// CleanupOldHourlySnapshots removes hourly snapshots older than the retention period
func (db *DB) CleanupOldHourlySnapshots(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.conn.Exec(`
		DELETE FROM usage_snapshots
		WHERE period_type = 'hourly' AND period_start < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanupOldDailySnapshots removes daily snapshots older than the retention period
func (db *DB) CleanupOldDailySnapshots(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.conn.Exec(`
		DELETE FROM usage_snapshots
		WHERE period_type = 'daily' AND period_start < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetUsageSummaryForAllOrgs returns usage summary for all organizations
func (db *DB) GetUsageSummaryForAllOrgs() ([]map[string]interface{}, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	rows, err := db.conn.Query(`
		SELECT o.id, o.name, o.plan_id,
		       COALESCE(SUM(u.bandwidth_bytes), 0) as bandwidth,
		       COALESCE(SUM(u.tunnel_seconds), 0) as tunnel_seconds,
		       COALESCE(SUM(u.request_count), 0) as requests,
		       COALESCE(MAX(u.peak_concurrent_tunnels), 0) as peak_concurrent
		FROM organizations o
		LEFT JOIN usage_snapshots u ON o.id = u.org_id 
			AND u.period_start >= ? AND u.period_start < ?
		GROUP BY o.id, o.name, o.plan_id
		ORDER BY bandwidth DESC
	`, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage summary: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var orgID, orgName string
		var planID sql.NullString
		var bandwidth, tunnelSeconds, requests int64
		var peakConcurrent int

		err := rows.Scan(&orgID, &orgName, &planID, &bandwidth, &tunnelSeconds, &requests, &peakConcurrent)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage summary: %w", err)
		}

		result := map[string]interface{}{
			"orgId":                 orgID,
			"orgName":               orgName,
			"bandwidthBytes":        bandwidth,
			"tunnelSeconds":         tunnelSeconds,
			"requestCount":          requests,
			"peakConcurrentTunnels": peakConcurrent,
		}
		if planID.Valid {
			result["planId"] = planID.String
		}
		results = append(results, result)
	}

	return results, rows.Err()
}
