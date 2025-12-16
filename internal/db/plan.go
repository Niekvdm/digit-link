package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Plan represents a subscription plan with quota limits
type Plan struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	BandwidthBytesMonthly *int64    `json:"bandwidthBytesMonthly,omitempty"`
	TunnelHoursMonthly    *int64    `json:"tunnelHoursMonthly,omitempty"`
	ConcurrentTunnelsMax  *int      `json:"concurrentTunnelsMax,omitempty"`
	RequestsMonthly       *int64    `json:"requestsMonthly,omitempty"`
	OverageAllowedPercent int       `json:"overageAllowedPercent"`
	GracePeriodHours      int       `json:"gracePeriodHours"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// CreatePlanInput holds the input for creating a plan
type CreatePlanInput struct {
	Name                  string `json:"name"`
	BandwidthBytesMonthly *int64 `json:"bandwidthBytesMonthly,omitempty"`
	TunnelHoursMonthly    *int64 `json:"tunnelHoursMonthly,omitempty"`
	ConcurrentTunnelsMax  *int   `json:"concurrentTunnelsMax,omitempty"`
	RequestsMonthly       *int64 `json:"requestsMonthly,omitempty"`
	OverageAllowedPercent int    `json:"overageAllowedPercent"`
	GracePeriodHours      int    `json:"gracePeriodHours"`
}

// CreatePlan creates a new plan
func (db *DB) CreatePlan(input CreatePlanInput) (*Plan, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO plans (
			id, name, bandwidth_bytes_monthly, tunnel_hours_monthly,
			concurrent_tunnels_max, requests_monthly, overage_allowed_percent,
			grace_period_hours, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, input.Name, input.BandwidthBytesMonthly, input.TunnelHoursMonthly,
		input.ConcurrentTunnelsMax, input.RequestsMonthly, input.OverageAllowedPercent,
		input.GracePeriodHours, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	return &Plan{
		ID:                    id,
		Name:                  input.Name,
		BandwidthBytesMonthly: input.BandwidthBytesMonthly,
		TunnelHoursMonthly:    input.TunnelHoursMonthly,
		ConcurrentTunnelsMax:  input.ConcurrentTunnelsMax,
		RequestsMonthly:       input.RequestsMonthly,
		OverageAllowedPercent: input.OverageAllowedPercent,
		GracePeriodHours:      input.GracePeriodHours,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

// GetPlan retrieves a plan by ID
func (db *DB) GetPlan(id string) (*Plan, error) {
	plan := &Plan{}
	var bandwidthBytes, tunnelHours, requests sql.NullInt64
	var concurrentTunnels sql.NullInt32

	err := db.conn.QueryRow(`
		SELECT id, name, bandwidth_bytes_monthly, tunnel_hours_monthly,
		       concurrent_tunnels_max, requests_monthly, overage_allowed_percent,
		       grace_period_hours, created_at, updated_at
		FROM plans WHERE id = ?
	`, id).Scan(
		&plan.ID, &plan.Name, &bandwidthBytes, &tunnelHours,
		&concurrentTunnels, &requests, &plan.OverageAllowedPercent,
		&plan.GracePeriodHours, &plan.CreatedAt, &plan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	if bandwidthBytes.Valid {
		plan.BandwidthBytesMonthly = &bandwidthBytes.Int64
	}
	if tunnelHours.Valid {
		plan.TunnelHoursMonthly = &tunnelHours.Int64
	}
	if concurrentTunnels.Valid {
		v := int(concurrentTunnels.Int32)
		plan.ConcurrentTunnelsMax = &v
	}
	if requests.Valid {
		plan.RequestsMonthly = &requests.Int64
	}

	return plan, nil
}

// GetPlanByName retrieves a plan by name
func (db *DB) GetPlanByName(name string) (*Plan, error) {
	plan := &Plan{}
	var bandwidthBytes, tunnelHours, requests sql.NullInt64
	var concurrentTunnels sql.NullInt32

	err := db.conn.QueryRow(`
		SELECT id, name, bandwidth_bytes_monthly, tunnel_hours_monthly,
		       concurrent_tunnels_max, requests_monthly, overage_allowed_percent,
		       grace_period_hours, created_at, updated_at
		FROM plans WHERE name = ?
	`, name).Scan(
		&plan.ID, &plan.Name, &bandwidthBytes, &tunnelHours,
		&concurrentTunnels, &requests, &plan.OverageAllowedPercent,
		&plan.GracePeriodHours, &plan.CreatedAt, &plan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	if bandwidthBytes.Valid {
		plan.BandwidthBytesMonthly = &bandwidthBytes.Int64
	}
	if tunnelHours.Valid {
		plan.TunnelHoursMonthly = &tunnelHours.Int64
	}
	if concurrentTunnels.Valid {
		v := int(concurrentTunnels.Int32)
		plan.ConcurrentTunnelsMax = &v
	}
	if requests.Valid {
		plan.RequestsMonthly = &requests.Int64
	}

	return plan, nil
}

// ListPlans returns all plans
func (db *DB) ListPlans() ([]*Plan, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, bandwidth_bytes_monthly, tunnel_hours_monthly,
		       concurrent_tunnels_max, requests_monthly, overage_allowed_percent,
		       grace_period_hours, created_at, updated_at
		FROM plans ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan := &Plan{}
		var bandwidthBytes, tunnelHours, requests sql.NullInt64
		var concurrentTunnels sql.NullInt32

		err := rows.Scan(
			&plan.ID, &plan.Name, &bandwidthBytes, &tunnelHours,
			&concurrentTunnels, &requests, &plan.OverageAllowedPercent,
			&plan.GracePeriodHours, &plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}

		if bandwidthBytes.Valid {
			plan.BandwidthBytesMonthly = &bandwidthBytes.Int64
		}
		if tunnelHours.Valid {
			plan.TunnelHoursMonthly = &tunnelHours.Int64
		}
		if concurrentTunnels.Valid {
			v := int(concurrentTunnels.Int32)
			plan.ConcurrentTunnelsMax = &v
		}
		if requests.Valid {
			plan.RequestsMonthly = &requests.Int64
		}

		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

// UpdatePlan updates an existing plan
func (db *DB) UpdatePlan(id string, input CreatePlanInput) (*Plan, error) {
	now := time.Now()

	result, err := db.conn.Exec(`
		UPDATE plans SET
			name = ?,
			bandwidth_bytes_monthly = ?,
			tunnel_hours_monthly = ?,
			concurrent_tunnels_max = ?,
			requests_monthly = ?,
			overage_allowed_percent = ?,
			grace_period_hours = ?,
			updated_at = ?
		WHERE id = ?
	`, input.Name, input.BandwidthBytesMonthly, input.TunnelHoursMonthly,
		input.ConcurrentTunnelsMax, input.RequestsMonthly, input.OverageAllowedPercent,
		input.GracePeriodHours, now, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return nil, nil
	}

	return db.GetPlan(id)
}

// DeletePlan deletes a plan (fails if any organizations are using it)
func (db *DB) DeletePlan(id string) error {
	// Check if any organizations are using this plan
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM organizations WHERE plan_id = ?
	`, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check plan usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete plan: %d organizations are using it", count)
	}

	_, err = db.conn.Exec(`DELETE FROM plans WHERE id = ?`, id)
	return err
}

// CountPlans returns the total number of plans
func (db *DB) CountPlans() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM plans`).Scan(&count)
	return count, err
}

// GetOrganizationsUsingPlan returns all organizations using a specific plan
func (db *DB) GetOrganizationsUsingPlan(planID string) ([]*Organization, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, plan_id, COALESCE(require_totp, 0), created_at
		FROM organizations WHERE plan_id = ?
		ORDER BY name
	`, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations using plan: %w", err)
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		var planID sql.NullString
		err := rows.Scan(&org.ID, &org.Name, &planID, &org.RequireTOTP, &org.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		if planID.Valid {
			org.PlanID = &planID.String
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}

// GetPlanForOrganization retrieves the plan for an organization
func (db *DB) GetPlanForOrganization(orgID string) (*Plan, error) {
	org, err := db.GetOrganizationByID(orgID)
	if err != nil {
		return nil, err
	}
	if org == nil || org.PlanID == nil {
		return nil, nil
	}
	return db.GetPlan(*org.PlanID)
}
