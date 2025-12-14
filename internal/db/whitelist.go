package db

import (
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
)

// WhitelistEntry represents an IP whitelist entry
type WhitelistEntry struct {
	ID          string    `json:"id"`
	IPRange     string    `json:"ipRange"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AccountWhitelistEntry represents an account-specific IP whitelist entry
type AccountWhitelistEntry struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"accountId"`
	IPRange     string    `json:"ipRange"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AddGlobalWhitelist adds an IP range to the global whitelist
func (db *DB) AddGlobalWhitelist(ipRange, description, createdBy string) (*WhitelistEntry, error) {
	// Validate CIDR or single IP
	if err := validateIPRange(ipRange); err != nil {
		return nil, fmt.Errorf("invalid IP range: %w", err)
	}

	id := uuid.New().String()
	now := time.Now()

	var createdByPtr *string
	if createdBy != "" {
		createdByPtr = &createdBy
	}

	_, err := db.conn.Exec(`
		INSERT INTO global_whitelist (id, ip_range, description, created_by, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, ipRange, description, createdByPtr, now)
	if err != nil {
		return nil, fmt.Errorf("failed to add whitelist entry: %w", err)
	}

	return &WhitelistEntry{
		ID:          id,
		IPRange:     ipRange,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}, nil
}

// ListGlobalWhitelist returns all global whitelist entries
func (db *DB) ListGlobalWhitelist() ([]*WhitelistEntry, error) {
	rows, err := db.conn.Query(`
		SELECT id, ip_range, description, created_by, created_at
		FROM global_whitelist ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list whitelist: %w", err)
	}
	defer rows.Close()

	var entries []*WhitelistEntry
	for rows.Next() {
		entry := &WhitelistEntry{}
		var description, createdBy sql.NullString

		err := rows.Scan(&entry.ID, &entry.IPRange, &description, &createdBy, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan whitelist entry: %w", err)
		}

		if description.Valid {
			entry.Description = description.String
		}
		if createdBy.Valid {
			entry.CreatedBy = createdBy.String
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetGlobalWhitelistEntry retrieves a specific whitelist entry
func (db *DB) GetGlobalWhitelistEntry(id string) (*WhitelistEntry, error) {
	entry := &WhitelistEntry{}
	var description, createdBy sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, ip_range, description, created_by, created_at
		FROM global_whitelist WHERE id = ?
	`, id).Scan(&entry.ID, &entry.IPRange, &description, &createdBy, &entry.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelist entry: %w", err)
	}

	if description.Valid {
		entry.Description = description.String
	}
	if createdBy.Valid {
		entry.CreatedBy = createdBy.String
	}

	return entry, nil
}

// DeleteGlobalWhitelist removes an IP range from the global whitelist
func (db *DB) DeleteGlobalWhitelist(id string) error {
	result, err := db.conn.Exec(`DELETE FROM global_whitelist WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete whitelist entry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("whitelist entry not found")
	}

	return nil
}

// AddAccountWhitelist adds an IP range to an account's whitelist
func (db *DB) AddAccountWhitelist(accountID, ipRange, description string) (*AccountWhitelistEntry, error) {
	if err := validateIPRange(ipRange); err != nil {
		return nil, fmt.Errorf("invalid IP range: %w", err)
	}

	id := uuid.New().String()
	now := time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO account_whitelist (id, account_id, ip_range, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, accountID, ipRange, description, now)
	if err != nil {
		return nil, fmt.Errorf("failed to add account whitelist entry: %w", err)
	}

	return &AccountWhitelistEntry{
		ID:          id,
		AccountID:   accountID,
		IPRange:     ipRange,
		Description: description,
		CreatedAt:   now,
	}, nil
}

// ListAccountWhitelist returns all whitelist entries for an account
func (db *DB) ListAccountWhitelist(accountID string) ([]*AccountWhitelistEntry, error) {
	rows, err := db.conn.Query(`
		SELECT id, account_id, ip_range, description, created_at
		FROM account_whitelist WHERE account_id = ? ORDER BY created_at DESC
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list account whitelist: %w", err)
	}
	defer rows.Close()

	var entries []*AccountWhitelistEntry
	for rows.Next() {
		entry := &AccountWhitelistEntry{}
		var description sql.NullString

		err := rows.Scan(&entry.ID, &entry.AccountID, &entry.IPRange, &description, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account whitelist entry: %w", err)
		}

		if description.Valid {
			entry.Description = description.String
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// DeleteAccountWhitelist removes an IP range from an account's whitelist
func (db *DB) DeleteAccountWhitelist(id string) error {
	result, err := db.conn.Exec(`DELETE FROM account_whitelist WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete account whitelist entry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("account whitelist entry not found")
	}

	return nil
}

// IsIPWhitelisted checks if an IP is in the global whitelist
func (db *DB) IsIPWhitelisted(ipStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	entries, err := db.ListGlobalWhitelist()
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if matchesIPRange(ip, entry.IPRange) {
			return true, nil
		}
	}

	return false, nil
}

// IsIPWhitelistedForAccount checks if an IP is whitelisted for a specific account
// It checks both global whitelist and account-specific whitelist
func (db *DB) IsIPWhitelistedForAccount(ipStr, accountID string) (bool, error) {
	// First check global whitelist
	whitelisted, err := db.IsIPWhitelisted(ipStr)
	if err != nil {
		return false, err
	}
	if whitelisted {
		return true, nil
	}

	// Then check account-specific whitelist
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	entries, err := db.ListAccountWhitelist(accountID)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if matchesIPRange(ip, entry.IPRange) {
			return true, nil
		}
	}

	return false, nil
}

// CountGlobalWhitelist returns the number of global whitelist entries
func (db *DB) CountGlobalWhitelist() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM global_whitelist`).Scan(&count)
	return count, err
}

// validateIPRange validates an IP address or CIDR range
func validateIPRange(ipRange string) error {
	// Try parsing as CIDR
	_, _, err := net.ParseCIDR(ipRange)
	if err == nil {
		return nil
	}

	// Try parsing as single IP
	ip := net.ParseIP(ipRange)
	if ip != nil {
		return nil
	}

	return fmt.Errorf("invalid IP address or CIDR range")
}

// matchesIPRange checks if an IP matches a given IP range (single IP or CIDR)
func matchesIPRange(ip net.IP, ipRange string) bool {
	// Try parsing as CIDR
	_, network, err := net.ParseCIDR(ipRange)
	if err == nil {
		return network.Contains(ip)
	}

	// Try parsing as single IP
	rangeIP := net.ParseIP(ipRange)
	if rangeIP != nil {
		return ip.Equal(rangeIP)
	}

	return false
}
