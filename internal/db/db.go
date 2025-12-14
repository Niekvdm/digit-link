package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
	CREATE TABLE IF NOT EXISTS accounts (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		token_hash TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used TIMESTAMP,
		active BOOLEAN DEFAULT TRUE
	);

	CREATE TABLE IF NOT EXISTS global_whitelist (
		id TEXT PRIMARY KEY,
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
		account_id TEXT NOT NULL,
		subdomain TEXT NOT NULL,
		client_ip TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		closed_at TIMESTAMP,
		bytes_sent INTEGER DEFAULT 0,
		bytes_received INTEGER DEFAULT 0,
		FOREIGN KEY(account_id) REFERENCES accounts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username);
	CREATE INDEX IF NOT EXISTS idx_accounts_token_hash ON accounts(token_hash);
	CREATE INDEX IF NOT EXISTS idx_tunnels_account_id ON tunnels(account_id);
	CREATE INDEX IF NOT EXISTS idx_tunnels_subdomain ON tunnels(subdomain);
	CREATE INDEX IF NOT EXISTS idx_global_whitelist_ip ON global_whitelist(ip_range);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetDBPath returns the default database path from environment or default
func GetDBPath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return "data/digit-link.db"
}
