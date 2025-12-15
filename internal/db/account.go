package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Account represents a user account in the system
type Account struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	TokenHash    string     `json:"-"` // Never expose hash
	PasswordHash string     `json:"-"` // Never expose hash
	TOTPSecret   string     `json:"-"` // Never expose secret
	TOTPEnabled  bool       `json:"totpEnabled"`
	IsAdmin      bool       `json:"isAdmin"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsed     *time.Time `json:"lastUsed,omitempty"`
	Active       bool       `json:"active"`
}

// CreateAccount creates a new account with the given username and token hash
func (db *DB) CreateAccount(username, tokenHash string, isAdmin bool) (*Account, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO accounts (id, username, token_hash, is_admin, created_at, active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, username, tokenHash, isAdmin, now, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &Account{
		ID:        id,
		Username:  username,
		TokenHash: tokenHash,
		IsAdmin:   isAdmin,
		CreatedAt: now,
		Active:    true,
	}, nil
}

// GetAccountByID retrieves an account by its ID
func (db *DB) GetAccountByID(id string) (*Account, error) {
	account := &Account{}
	var lastUsed sql.NullTime
	var passwordHash, totpSecret sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, username, token_hash, password_hash, totp_secret, totp_enabled, is_admin, created_at, last_used, active
		FROM accounts WHERE id = ?
	`, id).Scan(
		&account.ID, &account.Username, &account.TokenHash,
		&passwordHash, &totpSecret, &account.TOTPEnabled,
		&account.IsAdmin, &account.CreatedAt, &lastUsed, &account.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if lastUsed.Valid {
		account.LastUsed = &lastUsed.Time
	}
	if passwordHash.Valid {
		account.PasswordHash = passwordHash.String
	}
	if totpSecret.Valid {
		account.TOTPSecret = totpSecret.String
	}

	return account, nil
}

// GetAccountByTokenHash retrieves an account by its token hash
func (db *DB) GetAccountByTokenHash(tokenHash string) (*Account, error) {
	account := &Account{}
	var lastUsed sql.NullTime
	var passwordHash, totpSecret sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, username, token_hash, password_hash, totp_secret, totp_enabled, is_admin, created_at, last_used, active
		FROM accounts WHERE token_hash = ? AND active = TRUE
	`, tokenHash).Scan(
		&account.ID, &account.Username, &account.TokenHash,
		&passwordHash, &totpSecret, &account.TOTPEnabled,
		&account.IsAdmin, &account.CreatedAt, &lastUsed, &account.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if lastUsed.Valid {
		account.LastUsed = &lastUsed.Time
	}
	if passwordHash.Valid {
		account.PasswordHash = passwordHash.String
	}
	if totpSecret.Valid {
		account.TOTPSecret = totpSecret.String
	}

	return account, nil
}

// GetAccountByUsername retrieves an account by its username
func (db *DB) GetAccountByUsername(username string) (*Account, error) {
	account := &Account{}
	var lastUsed sql.NullTime
	var passwordHash, totpSecret sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, username, token_hash, password_hash, totp_secret, totp_enabled, is_admin, created_at, last_used, active
		FROM accounts WHERE username = ?
	`, username).Scan(
		&account.ID, &account.Username, &account.TokenHash,
		&passwordHash, &totpSecret, &account.TOTPEnabled,
		&account.IsAdmin, &account.CreatedAt, &lastUsed, &account.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if lastUsed.Valid {
		account.LastUsed = &lastUsed.Time
	}
	if passwordHash.Valid {
		account.PasswordHash = passwordHash.String
	}
	if totpSecret.Valid {
		account.TOTPSecret = totpSecret.String
	}

	return account, nil
}

// ListAccounts returns all accounts
func (db *DB) ListAccounts() ([]*Account, error) {
	rows, err := db.conn.Query(`
		SELECT id, username, token_hash, password_hash, totp_secret, totp_enabled, is_admin, created_at, last_used, active
		FROM accounts ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		account := &Account{}
		var lastUsed sql.NullTime
		var passwordHash, totpSecret sql.NullString

		err := rows.Scan(
			&account.ID, &account.Username, &account.TokenHash,
			&passwordHash, &totpSecret, &account.TOTPEnabled,
			&account.IsAdmin, &account.CreatedAt, &lastUsed, &account.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		if lastUsed.Valid {
			account.LastUsed = &lastUsed.Time
		}
		if passwordHash.Valid {
			account.PasswordHash = passwordHash.String
		}
		if totpSecret.Valid {
			account.TOTPSecret = totpSecret.String
		}

		accounts = append(accounts, account)
	}

	return accounts, rows.Err()
}

// UpdateAccountLastUsed updates the last_used timestamp for an account
func (db *DB) UpdateAccountLastUsed(id string) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET last_used = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

// UpdateAccountToken updates the token hash for an account
func (db *DB) UpdateAccountToken(id, tokenHash string) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET token_hash = ? WHERE id = ?
	`, tokenHash, id)
	return err
}

// UpdateAccountPassword updates the password hash for an account
func (db *DB) UpdateAccountPassword(id, passwordHash string) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET password_hash = ? WHERE id = ?
	`, passwordHash, id)
	return err
}

// UpdateAccountTOTP updates the TOTP secret and enabled status for an account
func (db *DB) UpdateAccountTOTP(id, totpSecret string, enabled bool) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET totp_secret = ?, totp_enabled = ? WHERE id = ?
	`, totpSecret, enabled, id)
	return err
}

// CreateAccountWithPassword creates a new account with username, password, and token
func (db *DB) CreateAccountWithPassword(username, tokenHash, passwordHash string, isAdmin bool) (*Account, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO accounts (id, username, token_hash, password_hash, is_admin, created_at, active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, username, tokenHash, passwordHash, isAdmin, now, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &Account{
		ID:           id,
		Username:     username,
		TokenHash:    tokenHash,
		PasswordHash: passwordHash,
		IsAdmin:      isAdmin,
		CreatedAt:    now,
		Active:       true,
	}, nil
}

// DeactivateAccount deactivates an account (soft delete)
func (db *DB) DeactivateAccount(id string) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET active = FALSE WHERE id = ?
	`, id)
	return err
}

// ActivateAccount activates an account
func (db *DB) ActivateAccount(id string) error {
	_, err := db.conn.Exec(`
		UPDATE accounts SET active = TRUE WHERE id = ?
	`, id)
	return err
}

// DeleteAccount permanently deletes an account
func (db *DB) DeleteAccount(id string) error {
	_, err := db.conn.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

// CountAccounts returns the total number of accounts
func (db *DB) CountAccounts() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&count)
	return count, err
}

// CountActiveAccounts returns the number of active accounts
func (db *DB) CountActiveAccounts() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM accounts WHERE active = TRUE`).Scan(&count)
	return count, err
}

// HasAdminAccount returns true if at least one admin account exists
func (db *DB) HasAdminAccount() (bool, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM accounts WHERE is_admin = TRUE AND active = TRUE`).Scan(&count)
	return count > 0, err
}
