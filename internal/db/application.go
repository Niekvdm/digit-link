package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuthMode represents the authentication mode for an application
type AuthMode string

const (
	AuthModeInherit  AuthMode = "inherit"
	AuthModeDisabled AuthMode = "disabled"
	AuthModeCustom   AuthMode = "custom"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeBasic  AuthType = "basic"
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeOIDC   AuthType = "oidc"
)

// Application represents a persistent application with auth policies
type Application struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"orgId"`
	Subdomain string    `json:"subdomain"`
	Name      string    `json:"name"`
	AuthMode  AuthMode  `json:"authMode"`
	AuthType  AuthType  `json:"authType,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateApplication creates a new application
func (db *DB) CreateApplication(orgID, subdomain, name string) (*Application, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO applications (id, org_id, subdomain, name, auth_mode, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, orgID, subdomain, name, AuthModeInherit, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	return &Application{
		ID:        id,
		OrgID:     orgID,
		Subdomain: subdomain,
		Name:      name,
		AuthMode:  AuthModeInherit,
		CreatedAt: now,
	}, nil
}

// GetApplicationByID retrieves an application by its ID
func (db *DB) GetApplicationByID(id string) (*Application, error) {
	app := &Application{}
	var name, authType sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, org_id, subdomain, name, auth_mode, auth_type, created_at
		FROM applications WHERE id = ?
	`, id).Scan(&app.ID, &app.OrgID, &app.Subdomain, &name, &app.AuthMode, &authType, &app.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	if name.Valid {
		app.Name = name.String
	}
	if authType.Valid {
		app.AuthType = AuthType(authType.String)
	}

	return app, nil
}

// GetApplicationBySubdomain retrieves an application by its subdomain
func (db *DB) GetApplicationBySubdomain(subdomain string) (*Application, error) {
	app := &Application{}
	var name, authType sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, org_id, subdomain, name, auth_mode, auth_type, created_at
		FROM applications WHERE subdomain = ?
	`, subdomain).Scan(&app.ID, &app.OrgID, &app.Subdomain, &name, &app.AuthMode, &authType, &app.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	if name.Valid {
		app.Name = name.String
	}
	if authType.Valid {
		app.AuthType = AuthType(authType.String)
	}

	return app, nil
}

// ListApplicationsByOrg returns all applications for an organization
func (db *DB) ListApplicationsByOrg(orgID string) ([]*Application, error) {
	rows, err := db.conn.Query(`
		SELECT id, org_id, subdomain, name, auth_mode, auth_type, created_at
		FROM applications WHERE org_id = ? ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}
	defer rows.Close()

	var apps []*Application
	for rows.Next() {
		app := &Application{}
		var name, authType sql.NullString

		err := rows.Scan(&app.ID, &app.OrgID, &app.Subdomain, &name, &app.AuthMode, &authType, &app.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application: %w", err)
		}

		if name.Valid {
			app.Name = name.String
		}
		if authType.Valid {
			app.AuthType = AuthType(authType.String)
		}

		apps = append(apps, app)
	}

	return apps, rows.Err()
}

// ListAllApplications returns all applications
func (db *DB) ListAllApplications() ([]*Application, error) {
	rows, err := db.conn.Query(`
		SELECT id, org_id, subdomain, name, auth_mode, auth_type, created_at
		FROM applications ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}
	defer rows.Close()

	var apps []*Application
	for rows.Next() {
		app := &Application{}
		var name, authType sql.NullString

		err := rows.Scan(&app.ID, &app.OrgID, &app.Subdomain, &name, &app.AuthMode, &authType, &app.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application: %w", err)
		}

		if name.Valid {
			app.Name = name.String
		}
		if authType.Valid {
			app.AuthType = AuthType(authType.String)
		}

		apps = append(apps, app)
	}

	return apps, rows.Err()
}

// UpdateApplication updates an application's name and auth settings
func (db *DB) UpdateApplication(id, name string, authMode AuthMode, authType AuthType) error {
	var authTypeStr *string
	if authType != "" {
		s := string(authType)
		authTypeStr = &s
	}

	_, err := db.conn.Exec(`
		UPDATE applications 
		SET name = ?, auth_mode = ?, auth_type = ?
		WHERE id = ?
	`, name, authMode, authTypeStr, id)
	return err
}

// UpdateApplicationAuthMode updates only the auth mode
func (db *DB) UpdateApplicationAuthMode(id string, authMode AuthMode) error {
	_, err := db.conn.Exec(`
		UPDATE applications SET auth_mode = ? WHERE id = ?
	`, authMode, id)
	return err
}

// DeleteApplication deletes an application
func (db *DB) DeleteApplication(id string) error {
	_, err := db.conn.Exec(`DELETE FROM applications WHERE id = ?`, id)
	return err
}

// IsSubdomainAvailable checks if a subdomain is available for a new application
func (db *DB) IsSubdomainAvailable(subdomain string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM applications WHERE subdomain = ?
	`, subdomain).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// CountApplicationsByOrg returns the number of applications for an organization
func (db *DB) CountApplicationsByOrg(orgID string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM applications WHERE org_id = ?
	`, orgID).Scan(&count)
	return count, err
}

// UpdateApplicationSubdomain updates an application's subdomain
// Returns error if the new subdomain is already in use by another application
func (db *DB) UpdateApplicationSubdomain(id, newSubdomain string) error {
	// Check if subdomain is available (excluding the current app)
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM applications WHERE subdomain = ? AND id != ?
	`, newSubdomain, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check subdomain availability: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("subdomain '%s' is already in use", newSubdomain)
	}

	_, err = db.conn.Exec(`
		UPDATE applications SET subdomain = ? WHERE id = ?
	`, newSubdomain, id)
	if err != nil {
		return fmt.Errorf("failed to update subdomain: %w", err)
	}
	return nil
}

// UpdateApplicationFull updates all editable fields of an application
func (db *DB) UpdateApplicationFull(id, name, subdomain string, authMode AuthMode, authType AuthType) error {
	// First check subdomain availability if it's being changed
	app, err := db.GetApplicationByID(id)
	if err != nil {
		return err
	}
	if app == nil {
		return fmt.Errorf("application not found")
	}

	if subdomain != app.Subdomain {
		var count int
		err := db.conn.QueryRow(`
			SELECT COUNT(*) FROM applications WHERE subdomain = ? AND id != ?
		`, subdomain, id).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check subdomain availability: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("subdomain '%s' is already in use", subdomain)
		}
	}

	var authTypeStr *string
	if authType != "" {
		s := string(authType)
		authTypeStr = &s
	}

	_, err = db.conn.Exec(`
		UPDATE applications 
		SET name = ?, subdomain = ?, auth_mode = ?, auth_type = ?
		WHERE id = ?
	`, name, subdomain, authMode, authTypeStr, id)
	return err
}
