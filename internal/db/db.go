package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for better concurrency handling
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// initSchema creates the database tables if they don't exist
func (db *DB) initSchema() error {
	schema := `
	-- Organizations (new layer above accounts)
	CREATE TABLE IF NOT EXISTS organizations (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		token_hash TEXT NOT NULL,
		password_hash TEXT,
		totp_secret TEXT,
		totp_enabled BOOLEAN DEFAULT FALSE,
		is_admin BOOLEAN DEFAULT FALSE,
		org_id TEXT REFERENCES organizations(id),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used TIMESTAMP,
		active BOOLEAN DEFAULT TRUE
	);

	-- Legacy global whitelist (deprecated, kept for migration)
	CREATE TABLE IF NOT EXISTS global_whitelist (
		id TEXT PRIMARY KEY,
		ip_range TEXT NOT NULL,
		description TEXT,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(created_by) REFERENCES accounts(id)
	);

	-- Organization-level whitelist (replaces global_whitelist)
	CREATE TABLE IF NOT EXISTS org_whitelist (
		id TEXT PRIMARY KEY,
		org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		ip_range TEXT NOT NULL,
		description TEXT,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(created_by) REFERENCES accounts(id)
	);

	-- Application-level whitelist
	CREATE TABLE IF NOT EXISTS app_whitelist (
		id TEXT PRIMARY KEY,
		app_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
		ip_range TEXT NOT NULL,
		description TEXT,
		created_by TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(created_by) REFERENCES accounts(id)
	);

	CREATE TABLE IF NOT EXISTS account_whitelist (
		id TEXT PRIMARY KEY,
		account_id TEXT NOT NULL,
		ip_range TEXT NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tunnels (
		id TEXT PRIMARY KEY,
		account_id TEXT,
		subdomain TEXT NOT NULL,
		client_ip TEXT,
		app_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		closed_at TIMESTAMP,
		bytes_sent INTEGER DEFAULT 0,
		bytes_received INTEGER DEFAULT 0,
		FOREIGN KEY(account_id) REFERENCES accounts(id)
	);

	-- Persistent applications with auth policies
	CREATE TABLE IF NOT EXISTS applications (
		id TEXT PRIMARY KEY,
		org_id TEXT NOT NULL REFERENCES organizations(id),
		subdomain TEXT UNIQUE NOT NULL,
		name TEXT,
		auth_mode TEXT DEFAULT 'inherit',
		auth_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Organization-level auth policy (default for org)
	CREATE TABLE IF NOT EXISTS org_auth_policies (
		org_id TEXT PRIMARY KEY REFERENCES organizations(id),
		auth_type TEXT NOT NULL,
		api_key_enabled BOOLEAN DEFAULT FALSE,
		basic_user_hash TEXT,
		basic_pass_hash TEXT,
		basic_session_duration INTEGER,
		oidc_issuer_url TEXT,
		oidc_client_id TEXT,
		oidc_client_secret_enc TEXT,
		oidc_scopes TEXT,
		oidc_allowed_domains TEXT,
		oidc_required_claims TEXT
	);

	-- App-level auth policy (when mode=custom)
	CREATE TABLE IF NOT EXISTS app_auth_policies (
		app_id TEXT PRIMARY KEY REFERENCES applications(id),
		auth_type TEXT NOT NULL,
		api_key_enabled BOOLEAN DEFAULT FALSE,
		basic_user_hash TEXT,
		basic_pass_hash TEXT,
		basic_session_duration INTEGER,
		oidc_issuer_url TEXT,
		oidc_client_id TEXT,
		oidc_client_secret_enc TEXT,
		oidc_scopes TEXT,
		oidc_allowed_domains TEXT,
		oidc_required_claims TEXT
	);

	-- API keys (hashed, with metadata)
	-- key_type: 'account' for random subdomain access, 'app' for specific app access
	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		org_id TEXT REFERENCES organizations(id),
		app_id TEXT REFERENCES applications(id),
		key_type TEXT DEFAULT 'account',
		key_hash TEXT NOT NULL,
		key_prefix TEXT NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used TIMESTAMP,
		expires_at TIMESTAMP
	);

	-- OIDC sessions (SQLite-backed)
	CREATE TABLE IF NOT EXISTS auth_sessions (
		id TEXT PRIMARY KEY,
		app_id TEXT,
		org_id TEXT,
		user_email TEXT,
		user_claims TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL
	);

	-- Rate limiting state
	CREATE TABLE IF NOT EXISTS rate_limit_state (
		key TEXT PRIMARY KEY,
		count INTEGER DEFAULT 0,
		window_start TIMESTAMP,
		blocked_until TIMESTAMP
	);

	-- Audit log
	CREATE TABLE IF NOT EXISTS auth_audit_log (
		id TEXT PRIMARY KEY,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		org_id TEXT,
		app_id TEXT,
		auth_type TEXT,
		success BOOLEAN,
		failure_reason TEXT,
		source_ip TEXT,
		user_identity TEXT,
		key_id TEXT
	);

	-- Subscription plans with quota limits
	CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		bandwidth_bytes_monthly BIGINT,
		tunnel_hours_monthly BIGINT,
		concurrent_tunnels_max INTEGER,
		requests_monthly BIGINT,
		overage_allowed_percent INTEGER DEFAULT 0,
		grace_period_hours INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Usage snapshots for tiered retention (hourly/daily/monthly)
	CREATE TABLE IF NOT EXISTS usage_snapshots (
		id TEXT PRIMARY KEY,
		org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		period_type TEXT NOT NULL,
		period_start TIMESTAMP NOT NULL,
		bandwidth_bytes BIGINT DEFAULT 0,
		tunnel_seconds BIGINT DEFAULT 0,
		request_count BIGINT DEFAULT 0,
		peak_concurrent_tunnels INTEGER DEFAULT 0,
		UNIQUE(org_id, period_type, period_start)
	);

	CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username);
	CREATE INDEX IF NOT EXISTS idx_accounts_token_hash ON accounts(token_hash);
	CREATE INDEX IF NOT EXISTS idx_accounts_org_id ON accounts(org_id);
	CREATE INDEX IF NOT EXISTS idx_tunnels_account_id ON tunnels(account_id);
	CREATE INDEX IF NOT EXISTS idx_tunnels_subdomain ON tunnels(subdomain);
	CREATE INDEX IF NOT EXISTS idx_tunnels_app_id ON tunnels(app_id);
	CREATE INDEX IF NOT EXISTS idx_global_whitelist_ip ON global_whitelist(ip_range);
	CREATE INDEX IF NOT EXISTS idx_org_whitelist_org_id ON org_whitelist(org_id);
	CREATE INDEX IF NOT EXISTS idx_org_whitelist_ip ON org_whitelist(ip_range);
	CREATE INDEX IF NOT EXISTS idx_app_whitelist_app_id ON app_whitelist(app_id);
	CREATE INDEX IF NOT EXISTS idx_app_whitelist_ip ON app_whitelist(ip_range);
	CREATE INDEX IF NOT EXISTS idx_applications_subdomain ON applications(subdomain);
	CREATE INDEX IF NOT EXISTS idx_applications_org_id ON applications(org_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
	CREATE INDEX IF NOT EXISTS idx_api_keys_org_id ON api_keys(org_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_app_id ON api_keys(app_id);
	CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires ON auth_sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_auth_audit_log_timestamp ON auth_audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_auth_audit_log_org_id ON auth_audit_log(org_id);
	CREATE INDEX IF NOT EXISTS idx_auth_audit_log_app_id ON auth_audit_log(app_id);
	CREATE INDEX IF NOT EXISTS idx_usage_snapshots_org_id ON usage_snapshots(org_id);
	CREATE INDEX IF NOT EXISTS idx_usage_snapshots_period ON usage_snapshots(period_type, period_start);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations for existing databases
	return db.runMigrations()
}

// runMigrations adds new columns to existing databases
func (db *DB) runMigrations() error {
	// List of column migrations to run (table, column, definition)
	columnMigrations := []struct {
		table  string
		column string
		def    string
	}{
		{"accounts", "password_hash", "TEXT"},
		{"accounts", "totp_secret", "TEXT"},
		{"accounts", "totp_enabled", "BOOLEAN DEFAULT FALSE"},
		{"accounts", "org_id", "TEXT REFERENCES organizations(id)"},
		{"accounts", "is_org_admin", "BOOLEAN DEFAULT FALSE"},
		{"tunnels", "app_id", "TEXT"},
		{"tunnels", "request_count", "BIGINT DEFAULT 0"},
		{"api_keys", "key_type", "TEXT DEFAULT 'account'"},
		{"organizations", "require_totp", "BOOLEAN DEFAULT FALSE"},
		{"organizations", "plan_id", "TEXT REFERENCES plans(id)"},
		{"org_auth_policies", "api_key_enabled", "BOOLEAN DEFAULT FALSE"},
		{"app_auth_policies", "api_key_enabled", "BOOLEAN DEFAULT FALSE"},
	}

	for _, m := range columnMigrations {
		var count int
		err := db.conn.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`,
			m.table, m.column,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check column %s.%s: %w", m.table, m.column, err)
		}

		if count == 0 {
			query := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, m.table, m.column, m.def)
			if _, err := db.conn.Exec(query); err != nil {
				// Ignore "duplicate column" errors
				if !strings.Contains(err.Error(), "duplicate column") {
					return fmt.Errorf("migration failed for %s.%s: %w", m.table, m.column, err)
				}
			}
		}
	}

	return nil
}

// GetDBPath returns the default database path from environment or default
func GetDBPath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return "data/digit-link.db"
}
