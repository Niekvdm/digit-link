package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TunnelRecord represents a tunnel connection record
type TunnelRecord struct {
	ID            string     `json:"id"`
	AccountID     string     `json:"accountId"`
	Subdomain     string     `json:"subdomain"`
	ClientIP      string     `json:"clientIp,omitempty"`
	AppID         string     `json:"appId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	ClosedAt      *time.Time `json:"closedAt,omitempty"`
	BytesSent     int64      `json:"bytesSent"`
	BytesReceived int64      `json:"bytesReceived"`
	RequestCount  int64      `json:"requestCount"`
}

// CreateTunnel creates a new tunnel record
func (db *DB) CreateTunnel(accountID, subdomain, clientIP string) (*TunnelRecord, error) {
	id := uuid.New().String()
	now := time.Now()

	// Handle empty accountID as NULL
	var accountIDParam interface{}
	if accountID != "" {
		accountIDParam = accountID
	}

	_, err := db.conn.Exec(`
		INSERT INTO tunnels (id, account_id, subdomain, client_ip, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, accountIDParam, subdomain, clientIP, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel record: %w", err)
	}

	return &TunnelRecord{
		ID:        id,
		AccountID: accountID,
		Subdomain: subdomain,
		ClientIP:  clientIP,
		CreatedAt: now,
	}, nil
}

// CloseTunnel marks a tunnel as closed
func (db *DB) CloseTunnel(id string) error {
	_, err := db.conn.Exec(`
		UPDATE tunnels SET closed_at = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

// UpdateTunnelStats updates the bytes sent/received for a tunnel
func (db *DB) UpdateTunnelStats(id string, bytesSent, bytesReceived int64) error {
	_, err := db.conn.Exec(`
		UPDATE tunnels SET bytes_sent = bytes_sent + ?, bytes_received = bytes_received + ?
		WHERE id = ?
	`, bytesSent, bytesReceived, id)
	return err
}

// IncrementTunnelRequestCount increments the request count for a tunnel
func (db *DB) IncrementTunnelRequestCount(id string) error {
	_, err := db.conn.Exec(`
		UPDATE tunnels SET request_count = request_count + 1
		WHERE id = ?
	`, id)
	return err
}

// UpdateTunnelStatsWithRequests updates bytes and request count atomically
func (db *DB) UpdateTunnelStatsWithRequests(id string, bytesSent, bytesReceived int64, requests int64) error {
	_, err := db.conn.Exec(`
		UPDATE tunnels SET 
			bytes_sent = bytes_sent + ?, 
			bytes_received = bytes_received + ?,
			request_count = request_count + ?
		WHERE id = ?
	`, bytesSent, bytesReceived, requests, id)
	return err
}

// GetTunnel retrieves a tunnel record by ID
func (db *DB) GetTunnel(id string) (*TunnelRecord, error) {
	record := &TunnelRecord{}
	var closedAt sql.NullTime
	var clientIP sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, account_id, subdomain, client_ip, created_at, closed_at, bytes_sent, bytes_received
		FROM tunnels WHERE id = ?
	`, id).Scan(
		&record.ID, &record.AccountID, &record.Subdomain, &clientIP,
		&record.CreatedAt, &closedAt, &record.BytesSent, &record.BytesReceived,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel: %w", err)
	}

	if closedAt.Valid {
		record.ClosedAt = &closedAt.Time
	}
	if clientIP.Valid {
		record.ClientIP = clientIP.String
	}

	return record, nil
}

// ListActiveTunnels returns all currently active (not closed) tunnels
func (db *DB) ListActiveTunnels() ([]*TunnelRecord, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.account_id, t.subdomain, t.client_ip, t.created_at, t.closed_at, 
		       t.bytes_sent, t.bytes_received, a.username
		FROM tunnels t
		LEFT JOIN accounts a ON t.account_id = a.id
		WHERE t.closed_at IS NULL
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tunnels: %w", err)
	}
	defer rows.Close()

	var tunnels []*TunnelRecord
	for rows.Next() {
		record := &TunnelRecord{}
		var closedAt sql.NullTime
		var clientIP, username sql.NullString

		err := rows.Scan(
			&record.ID, &record.AccountID, &record.Subdomain, &clientIP,
			&record.CreatedAt, &closedAt, &record.BytesSent, &record.BytesReceived,
			&username,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tunnel: %w", err)
		}

		if closedAt.Valid {
			record.ClosedAt = &closedAt.Time
		}
		if clientIP.Valid {
			record.ClientIP = clientIP.String
		}

		tunnels = append(tunnels, record)
	}

	return tunnels, rows.Err()
}

// ListTunnelsForAccount returns all tunnels for a specific account
func (db *DB) ListTunnelsForAccount(accountID string) ([]*TunnelRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, account_id, subdomain, client_ip, created_at, closed_at, bytes_sent, bytes_received
		FROM tunnels WHERE account_id = ?
		ORDER BY created_at DESC
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tunnels for account: %w", err)
	}
	defer rows.Close()

	var tunnels []*TunnelRecord
	for rows.Next() {
		record := &TunnelRecord{}
		var closedAt sql.NullTime
		var clientIP sql.NullString

		err := rows.Scan(
			&record.ID, &record.AccountID, &record.Subdomain, &clientIP,
			&record.CreatedAt, &closedAt, &record.BytesSent, &record.BytesReceived,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tunnel: %w", err)
		}

		if closedAt.Valid {
			record.ClosedAt = &closedAt.Time
		}
		if clientIP.Valid {
			record.ClientIP = clientIP.String
		}

		tunnels = append(tunnels, record)
	}

	return tunnels, rows.Err()
}

// CountActiveTunnels returns the number of active tunnels
func (db *DB) CountActiveTunnels() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM tunnels WHERE closed_at IS NULL`).Scan(&count)
	return count, err
}

// GetTotalTunnelStats returns aggregate statistics for all tunnels
func (db *DB) GetTotalTunnelStats() (totalTunnels int, totalBytesSent, totalBytesReceived int64, err error) {
	err = db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(bytes_sent), 0), COALESCE(SUM(bytes_received), 0)
		FROM tunnels
	`).Scan(&totalTunnels, &totalBytesSent, &totalBytesReceived)
	return
}

// CleanupOldTunnels removes tunnel records older than the specified duration
func (db *DB) CleanupOldTunnels(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.conn.Exec(`
		DELETE FROM tunnels WHERE closed_at IS NOT NULL AND closed_at < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// UpdateTunnelAppID updates the app_id for a tunnel record
func (db *DB) UpdateTunnelAppID(id, appID string) error {
	_, err := db.conn.Exec(`UPDATE tunnels SET app_id = ? WHERE id = ?`, appID, id)
	return err
}

// ============================================
// App and Org specific tunnel methods
// ============================================

// TunnelStats holds aggregated statistics for tunnels
type TunnelStats struct {
	TotalConnections int   `json:"totalConnections"`
	ActiveCount      int   `json:"activeCount"`
	BytesSent        int64 `json:"bytesSent"`
	BytesReceived    int64 `json:"bytesReceived"`
	RequestCount     int64 `json:"requestCount"`
}

// GetTunnelStatsByApp returns statistics for a specific application
func (db *DB) GetTunnelStatsByApp(appID string) (*TunnelStats, error) {
	stats := &TunnelStats{}

	// Get total connections and bytes for this app
	err := db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(bytes_sent), 0), COALESCE(SUM(bytes_received), 0), COALESCE(SUM(request_count), 0)
		FROM tunnels WHERE app_id = ?
	`, appID).Scan(&stats.TotalConnections, &stats.BytesSent, &stats.BytesReceived, &stats.RequestCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel stats for app: %w", err)
	}

	// Get active count
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM tunnels WHERE app_id = ? AND closed_at IS NULL
	`, appID).Scan(&stats.ActiveCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tunnel count for app: %w", err)
	}

	return stats, nil
}

// GetTunnelStatsByOrg returns statistics for a specific organization
func (db *DB) GetTunnelStatsByOrg(orgID string) (*TunnelStats, error) {
	stats := &TunnelStats{}

	// Get total connections and bytes for all apps in this org
	err := db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(t.bytes_sent), 0), COALESCE(SUM(t.bytes_received), 0), COALESCE(SUM(t.request_count), 0)
		FROM tunnels t
		JOIN applications a ON t.app_id = a.id
		WHERE a.org_id = ?
	`, orgID).Scan(&stats.TotalConnections, &stats.BytesSent, &stats.BytesReceived, &stats.RequestCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel stats for org: %w", err)
	}

	// Get active count
	err = db.conn.QueryRow(`
		SELECT COUNT(*) 
		FROM tunnels t
		JOIN applications a ON t.app_id = a.id
		WHERE a.org_id = ? AND t.closed_at IS NULL
	`, orgID).Scan(&stats.ActiveCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tunnel count for org: %w", err)
	}

	return stats, nil
}

// ListActiveTunnelsByApp returns all active tunnels for a specific application
func (db *DB) ListActiveTunnelsByApp(appID string) ([]*TunnelRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, account_id, subdomain, client_ip, app_id, created_at, closed_at, bytes_sent, bytes_received
		FROM tunnels WHERE app_id = ? AND closed_at IS NULL
		ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tunnels for app: %w", err)
	}
	defer rows.Close()

	return scanTunnelRecords(rows)
}

// ListActiveTunnelsByOrg returns all active tunnels for an organization
func (db *DB) ListActiveTunnelsByOrg(orgID string) ([]*TunnelRecord, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.account_id, t.subdomain, t.client_ip, t.app_id, t.created_at, t.closed_at, t.bytes_sent, t.bytes_received
		FROM tunnels t
		JOIN applications a ON t.app_id = a.id
		WHERE a.org_id = ? AND t.closed_at IS NULL
		ORDER BY t.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active tunnels for org: %w", err)
	}
	defer rows.Close()

	return scanTunnelRecords(rows)
}

// CountActiveTunnelsByApp returns the number of active tunnels for an application
func (db *DB) CountActiveTunnelsByApp(appID string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM tunnels WHERE app_id = ? AND closed_at IS NULL
	`, appID).Scan(&count)
	return count, err
}

// CountActiveTunnelsByOrg returns the number of active tunnels for an organization
func (db *DB) CountActiveTunnelsByOrg(orgID string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) 
		FROM tunnels t
		JOIN applications a ON t.app_id = a.id
		WHERE a.org_id = ? AND t.closed_at IS NULL
	`, orgID).Scan(&count)
	return count, err
}

// Helper function to scan tunnel records
func scanTunnelRecords(rows *sql.Rows) ([]*TunnelRecord, error) {
	var tunnels []*TunnelRecord
	for rows.Next() {
		record := &TunnelRecord{}
		var closedAt sql.NullTime
		var clientIP, appID sql.NullString

		err := rows.Scan(
			&record.ID, &record.AccountID, &record.Subdomain, &clientIP, &appID,
			&record.CreatedAt, &closedAt, &record.BytesSent, &record.BytesReceived,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tunnel: %w", err)
		}

		if closedAt.Valid {
			record.ClosedAt = &closedAt.Time
		}
		if clientIP.Valid {
			record.ClientIP = clientIP.String
		}
		if appID.Valid {
			record.AppID = appID.String
		}

		tunnels = append(tunnels, record)
	}

	return tunnels, rows.Err()
}
