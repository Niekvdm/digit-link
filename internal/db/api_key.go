package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// APIKeyLength is the length of generated API keys in bytes
	APIKeyLength = 32
	// APIKeyPrefixLength is the length of the key prefix stored for identification
	APIKeyPrefixLength = 8
)

// APIKey represents an API key for authentication
type APIKey struct {
	ID          string     `json:"id"`
	OrgID       *string    `json:"orgId,omitempty"`
	AppID       *string    `json:"appId,omitempty"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"keyPrefix"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsed    *time.Time `json:"lastUsed,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

// GenerateAPIKey generates a new API key
// Returns the raw key (to show user once) and the key struct with hash
func GenerateAPIKey(orgID, appID *string, description string, expiresAt *time.Time) (rawKey string, key *APIKey, err error) {
	bytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	rawKey = hex.EncodeToString(bytes)
	keyHash := HashAPIKey(rawKey)
	keyPrefix := rawKey[:APIKeyPrefixLength]

	key = &APIKey{
		ID:          uuid.New().String(),
		OrgID:       orgID,
		AppID:       appID,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Description: description,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	return rawKey, key, nil
}

// HashAPIKey creates a SHA-256 hash of an API key
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CreateAPIKey stores a new API key in the database
func (db *DB) CreateAPIKey(key *APIKey) error {
	_, err := db.conn.Exec(`
		INSERT INTO api_keys (id, org_id, app_id, key_hash, key_prefix, description, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, key.ID, key.OrgID, key.AppID, key.KeyHash, key.KeyPrefix, key.Description, key.CreatedAt, key.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	return nil
}

// GetAPIKeyByID retrieves an API key by its ID
func (db *DB) GetAPIKeyByID(id string) (*APIKey, error) {
	key := &APIKey{}
	var orgID, appID, description sql.NullString
	var lastUsed, expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, org_id, app_id, key_hash, key_prefix, description, created_at, last_used, expires_at
		FROM api_keys WHERE id = ?
	`, id).Scan(
		&key.ID, &orgID, &appID, &key.KeyHash, &key.KeyPrefix, &description,
		&key.CreatedAt, &lastUsed, &expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if orgID.Valid {
		key.OrgID = &orgID.String
	}
	if appID.Valid {
		key.AppID = &appID.String
	}
	if description.Valid {
		key.Description = description.String
	}
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return key, nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (db *DB) GetAPIKeyByHash(keyHash string) (*APIKey, error) {
	key := &APIKey{}
	var orgID, appID, description sql.NullString
	var lastUsed, expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, org_id, app_id, key_hash, key_prefix, description, created_at, last_used, expires_at
		FROM api_keys WHERE key_hash = ?
	`, keyHash).Scan(
		&key.ID, &orgID, &appID, &key.KeyHash, &key.KeyPrefix, &description,
		&key.CreatedAt, &lastUsed, &expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if orgID.Valid {
		key.OrgID = &orgID.String
	}
	if appID.Valid {
		key.AppID = &appID.String
	}
	if description.Valid {
		key.Description = description.String
	}
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return key, nil
}

// ValidateAPIKey validates an API key and returns the key record if valid
func (db *DB) ValidateAPIKey(rawKey string) (*APIKey, error) {
	keyHash := HashAPIKey(rawKey)
	key, err := db.GetAPIKeyByHash(keyHash)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, nil
	}

	// Check if key is expired
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	return key, nil
}

// ListAPIKeysByOrg returns all API keys for an organization
func (db *DB) ListAPIKeysByOrg(orgID string) ([]*APIKey, error) {
	rows, err := db.conn.Query(`
		SELECT id, org_id, app_id, key_hash, key_prefix, description, created_at, last_used, expires_at
		FROM api_keys WHERE org_id = ? ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	return scanAPIKeys(rows)
}

// ListAPIKeysByApp returns all API keys for an application
func (db *DB) ListAPIKeysByApp(appID string) ([]*APIKey, error) {
	rows, err := db.conn.Query(`
		SELECT id, org_id, app_id, key_hash, key_prefix, description, created_at, last_used, expires_at
		FROM api_keys WHERE app_id = ? ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	return scanAPIKeys(rows)
}

// ListAPIKeysForAuth returns all API keys for an org or app (used for auth matching)
func (db *DB) ListAPIKeysForAuth(orgID, appID *string) ([]*APIKey, error) {
	var rows *sql.Rows
	var err error

	if appID != nil {
		// First try app-specific keys, then fall back to org keys
		rows, err = db.conn.Query(`
			SELECT id, org_id, app_id, key_hash, key_prefix, description, created_at, last_used, expires_at
			FROM api_keys 
			WHERE (app_id = ? OR (app_id IS NULL AND org_id = ?))
			AND (expires_at IS NULL OR expires_at > ?)
			ORDER BY app_id DESC NULLS LAST, created_at DESC
		`, *appID, orgID, time.Now())
	} else if orgID != nil {
		rows, err = db.conn.Query(`
			SELECT id, org_id, app_id, key_hash, key_prefix, description, created_at, last_used, expires_at
			FROM api_keys 
			WHERE org_id = ? AND app_id IS NULL
			AND (expires_at IS NULL OR expires_at > ?)
			ORDER BY created_at DESC
		`, *orgID, time.Now())
	} else {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list API keys for auth: %w", err)
	}
	defer rows.Close()

	return scanAPIKeys(rows)
}

func scanAPIKeys(rows *sql.Rows) ([]*APIKey, error) {
	var keys []*APIKey
	for rows.Next() {
		key := &APIKey{}
		var orgID, appID, description sql.NullString
		var lastUsed, expiresAt sql.NullTime

		err := rows.Scan(
			&key.ID, &orgID, &appID, &key.KeyHash, &key.KeyPrefix, &description,
			&key.CreatedAt, &lastUsed, &expiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if orgID.Valid {
			key.OrgID = &orgID.String
		}
		if appID.Valid {
			key.AppID = &appID.String
		}
		if description.Valid {
			key.Description = description.String
		}
		if lastUsed.Valid {
			key.LastUsed = &lastUsed.Time
		}
		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}

		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key
func (db *DB) UpdateAPIKeyLastUsed(id string) error {
	_, err := db.conn.Exec(`
		UPDATE api_keys SET last_used = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

// DeleteAPIKey deletes an API key
func (db *DB) DeleteAPIKey(id string) error {
	_, err := db.conn.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

// DeleteExpiredAPIKeys removes all expired API keys
func (db *DB) DeleteExpiredAPIKeys() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountAPIKeysByOrg returns the number of API keys for an organization
func (db *DB) CountAPIKeysByOrg(orgID string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM api_keys WHERE org_id = ?
	`, orgID).Scan(&count)
	return count, err
}
