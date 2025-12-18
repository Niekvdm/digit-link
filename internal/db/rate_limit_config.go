package db

import (
	"database/sql"
	"time"
)

// AppRateLimitConfig represents per-application rate limiting configuration
type AppRateLimitConfig struct {
	AppID                 string    `json:"appId"`
	Enabled               bool      `json:"enabled"`
	MaxAttempts           int       `json:"maxAttempts"`
	WindowDurationSeconds int       `json:"windowDurationSeconds"`
	BlockDurationSeconds  int       `json:"blockDurationSeconds"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

// DefaultRateLimitValues returns the default rate limit configuration values
func DefaultRateLimitValues() (maxAttempts, windowSeconds, blockSeconds int) {
	return 10, 900, 1800 // 10 attempts, 15 min window, 30 min block
}

// GetAppRateLimitConfig retrieves the rate limit configuration for an application
func (db *DB) GetAppRateLimitConfig(appID string) (*AppRateLimitConfig, error) {
	var config AppRateLimitConfig
	var updatedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT app_id, enabled, max_attempts, window_duration_seconds, block_duration_seconds, updated_at
		FROM app_rate_limit_config
		WHERE app_id = ?
	`, appID).Scan(
		&config.AppID,
		&config.Enabled,
		&config.MaxAttempts,
		&config.WindowDurationSeconds,
		&config.BlockDurationSeconds,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if updatedAt.Valid {
		config.UpdatedAt = updatedAt.Time
	}

	return &config, nil
}

// SetAppRateLimitConfig creates or updates the rate limit configuration for an application
func (db *DB) SetAppRateLimitConfig(config *AppRateLimitConfig) error {
	_, err := db.conn.Exec(`
		INSERT INTO app_rate_limit_config (app_id, enabled, max_attempts, window_duration_seconds, block_duration_seconds, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(app_id) DO UPDATE SET
			enabled = excluded.enabled,
			max_attempts = excluded.max_attempts,
			window_duration_seconds = excluded.window_duration_seconds,
			block_duration_seconds = excluded.block_duration_seconds,
			updated_at = CURRENT_TIMESTAMP
	`, config.AppID, config.Enabled, config.MaxAttempts, config.WindowDurationSeconds, config.BlockDurationSeconds)

	return err
}

// DeleteAppRateLimitConfig removes the rate limit configuration for an application
func (db *DB) DeleteAppRateLimitConfig(appID string) error {
	_, err := db.conn.Exec(`DELETE FROM app_rate_limit_config WHERE app_id = ?`, appID)
	return err
}

// ListAppRateLimitConfigs returns all rate limit configurations (for admin use)
func (db *DB) ListAppRateLimitConfigs() ([]*AppRateLimitConfig, error) {
	rows, err := db.conn.Query(`
		SELECT app_id, enabled, max_attempts, window_duration_seconds, block_duration_seconds, updated_at
		FROM app_rate_limit_config
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*AppRateLimitConfig
	for rows.Next() {
		var config AppRateLimitConfig
		var updatedAt sql.NullTime

		if err := rows.Scan(
			&config.AppID,
			&config.Enabled,
			&config.MaxAttempts,
			&config.WindowDurationSeconds,
			&config.BlockDurationSeconds,
			&updatedAt,
		); err != nil {
			return nil, err
		}

		if updatedAt.Valid {
			config.UpdatedAt = updatedAt.Time
		}

		configs = append(configs, &config)
	}

	return configs, rows.Err()
}
