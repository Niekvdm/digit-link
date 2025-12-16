package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Organization represents an organization that owns accounts and applications
type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PlanID      *string   `json:"planId,omitempty"`
	RequireTOTP bool      `json:"requireTotp"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateOrganization creates a new organization
func (db *DB) CreateOrganization(name string) (*Organization, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO organizations (id, name, require_totp, created_at)
		VALUES (?, ?, ?, ?)
	`, id, name, false, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return &Organization{
		ID:          id,
		Name:        name,
		PlanID:      nil,
		RequireTOTP: false,
		CreatedAt:   now,
	}, nil
}

// CreateOrganizationWithPlan creates a new organization with a plan
func (db *DB) CreateOrganizationWithPlan(name string, planID *string) (*Organization, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO organizations (id, name, plan_id, require_totp, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, name, planID, false, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return &Organization{
		ID:          id,
		Name:        name,
		PlanID:      planID,
		RequireTOTP: false,
		CreatedAt:   now,
	}, nil
}

// GetOrganizationByID retrieves an organization by its ID
func (db *DB) GetOrganizationByID(id string) (*Organization, error) {
	org := &Organization{}
	var planID sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, plan_id, COALESCE(require_totp, 0), created_at
		FROM organizations WHERE id = ?
	`, id).Scan(&org.ID, &org.Name, &planID, &org.RequireTOTP, &org.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if planID.Valid {
		org.PlanID = &planID.String
	}

	return org, nil
}

// GetOrganizationByName retrieves an organization by its name
func (db *DB) GetOrganizationByName(name string) (*Organization, error) {
	org := &Organization{}
	var planID sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, plan_id, COALESCE(require_totp, 0), created_at
		FROM organizations WHERE name = ?
	`, name).Scan(&org.ID, &org.Name, &planID, &org.RequireTOTP, &org.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if planID.Valid {
		org.PlanID = &planID.String
	}

	return org, nil
}

// ListOrganizations returns all organizations
func (db *DB) ListOrganizations() ([]*Organization, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, plan_id, COALESCE(require_totp, 0), created_at
		FROM organizations ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
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

// UpdateOrganization updates an organization's name
func (db *DB) UpdateOrganization(id, name string) error {
	_, err := db.conn.Exec(`
		UPDATE organizations SET name = ? WHERE id = ?
	`, name, id)
	return err
}

// UpdateOrganizationTOTPRequirement updates the TOTP requirement for an organization
func (db *DB) UpdateOrganizationTOTPRequirement(id string, requireTOTP bool) error {
	_, err := db.conn.Exec(`
		UPDATE organizations SET require_totp = ? WHERE id = ?
	`, requireTOTP, id)
	return err
}

// UpdateOrganizationPlan updates the plan for an organization
func (db *DB) UpdateOrganizationPlan(id string, planID *string) error {
	_, err := db.conn.Exec(`
		UPDATE organizations SET plan_id = ? WHERE id = ?
	`, planID, id)
	return err
}

// DeleteOrganization deletes an organization
func (db *DB) DeleteOrganization(id string) error {
	_, err := db.conn.Exec(`DELETE FROM organizations WHERE id = ?`, id)
	return err
}

// GetOrganizationByAccountID retrieves the organization for a given account
func (db *DB) GetOrganizationByAccountID(accountID string) (*Organization, error) {
	org := &Organization{}
	var planID sql.NullString

	err := db.conn.QueryRow(`
		SELECT o.id, o.name, o.plan_id, COALESCE(o.require_totp, 0), o.created_at
		FROM organizations o
		JOIN accounts a ON a.org_id = o.id
		WHERE a.id = ?
	`, accountID).Scan(&org.ID, &org.Name, &planID, &org.RequireTOTP, &org.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization by account: %w", err)
	}

	if planID.Valid {
		org.PlanID = &planID.String
	}

	return org, nil
}

// SetAccountOrganization sets the organization for an account (empty string to unlink)
func (db *DB) SetAccountOrganization(accountID, orgID string) error {
	var orgPtr interface{}
	if orgID == "" {
		orgPtr = nil // Set to NULL when unlinking
	} else {
		orgPtr = orgID
	}
	_, err := db.conn.Exec(`
		UPDATE accounts SET org_id = ? WHERE id = ?
	`, orgPtr, accountID)
	return err
}

// CountOrganizations returns the total number of organizations
func (db *DB) CountOrganizations() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM organizations`).Scan(&count)
	return count, err
}

// GetOrCreateDefaultOrganization gets or creates a default organization for migration
func (db *DB) GetOrCreateDefaultOrganization() (*Organization, error) {
	org, err := db.GetOrganizationByName("default")
	if err != nil {
		return nil, err
	}
	if org != nil {
		return org, nil
	}

	return db.CreateOrganization("default")
}
