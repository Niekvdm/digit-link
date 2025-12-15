package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Organization represents an organization that owns accounts and applications
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateOrganization creates a new organization
func (db *DB) CreateOrganization(name string) (*Organization, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO organizations (id, name, created_at)
		VALUES (?, ?, ?)
	`, id, name, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return &Organization{
		ID:        id,
		Name:      name,
		CreatedAt: now,
	}, nil
}

// GetOrganizationByID retrieves an organization by its ID
func (db *DB) GetOrganizationByID(id string) (*Organization, error) {
	org := &Organization{}

	err := db.conn.QueryRow(`
		SELECT id, name, created_at
		FROM organizations WHERE id = ?
	`, id).Scan(&org.ID, &org.Name, &org.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// GetOrganizationByName retrieves an organization by its name
func (db *DB) GetOrganizationByName(name string) (*Organization, error) {
	org := &Organization{}

	err := db.conn.QueryRow(`
		SELECT id, name, created_at
		FROM organizations WHERE name = ?
	`, name).Scan(&org.ID, &org.Name, &org.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// ListOrganizations returns all organizations
func (db *DB) ListOrganizations() ([]*Organization, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, created_at
		FROM organizations ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		err := rows.Scan(&org.ID, &org.Name, &org.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
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

// DeleteOrganization deletes an organization
func (db *DB) DeleteOrganization(id string) error {
	_, err := db.conn.Exec(`DELETE FROM organizations WHERE id = ?`, id)
	return err
}

// GetOrganizationByAccountID retrieves the organization for a given account
func (db *DB) GetOrganizationByAccountID(accountID string) (*Organization, error) {
	org := &Organization{}

	err := db.conn.QueryRow(`
		SELECT o.id, o.name, o.created_at
		FROM organizations o
		JOIN accounts a ON a.org_id = o.id
		WHERE a.id = ?
	`, accountID).Scan(&org.ID, &org.Name, &org.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization by account: %w", err)
	}

	return org, nil
}

// SetAccountOrganization sets the organization for an account
func (db *DB) SetAccountOrganization(accountID, orgID string) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET org_id = ? WHERE id = ?
	`, orgID, accountID)
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
