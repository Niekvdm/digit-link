package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditEvent represents an authentication audit log entry
type AuditEvent struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	OrgID         *string   `json:"orgId,omitempty"`
	AppID         *string   `json:"appId,omitempty"`
	AuthType      string    `json:"authType"`
	Success       bool      `json:"success"`
	FailureReason string    `json:"failureReason,omitempty"`
	SourceIP      string    `json:"sourceIp"`
	UserIdentity  string    `json:"userIdentity,omitempty"`
	KeyID         string    `json:"keyId,omitempty"`
}

// LogAuthEvent logs an authentication event
func (db *DB) LogAuthEvent(event *AuditEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	_, err := db.conn.Exec(`
		INSERT INTO auth_audit_log (
			id, timestamp, org_id, app_id, auth_type, success,
			failure_reason, source_ip, user_identity, key_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.Timestamp, event.OrgID, event.AppID, event.AuthType,
		event.Success, event.FailureReason, event.SourceIP, event.UserIdentity, event.KeyID)

	if err != nil {
		return fmt.Errorf("failed to log auth event: %w", err)
	}
	return nil
}

// LogAuthSuccess logs a successful authentication event
func (db *DB) LogAuthSuccess(orgID, appID *string, authType, sourceIP, userIdentity, keyID string) error {
	return db.LogAuthEvent(&AuditEvent{
		OrgID:        orgID,
		AppID:        appID,
		AuthType:     authType,
		Success:      true,
		SourceIP:     sourceIP,
		UserIdentity: userIdentity,
		KeyID:        keyID,
	})
}

// LogAuthFailure logs a failed authentication event
func (db *DB) LogAuthFailure(orgID, appID *string, authType, sourceIP, failureReason string) error {
	return db.LogAuthEvent(&AuditEvent{
		OrgID:         orgID,
		AppID:         appID,
		AuthType:      authType,
		Success:       false,
		FailureReason: failureReason,
		SourceIP:      sourceIP,
	})
}

// GetAuditEvents retrieves audit events with optional filtering
func (db *DB) GetAuditEvents(orgID, appID *string, limit, offset int) ([]*AuditEvent, error) {
	query := `
		SELECT id, timestamp, org_id, app_id, auth_type, success,
			failure_reason, source_ip, user_identity, key_id
		FROM auth_audit_log
		WHERE 1=1
	`
	args := []interface{}{}

	if orgID != nil {
		query += " AND org_id = ?"
		args = append(args, *orgID)
	}
	if appID != nil {
		query += " AND app_id = ?"
		args = append(args, *appID)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit events: %w", err)
	}
	defer rows.Close()

	return scanAuditEvents(rows)
}

// GetRecentAuditEvents retrieves recent audit events within a time window
func (db *DB) GetRecentAuditEvents(since time.Time, limit int) ([]*AuditEvent, error) {
	rows, err := db.conn.Query(`
		SELECT id, timestamp, org_id, app_id, auth_type, success,
			failure_reason, source_ip, user_identity, key_id
		FROM auth_audit_log
		WHERE timestamp > ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent audit events: %w", err)
	}
	defer rows.Close()

	return scanAuditEvents(rows)
}

// GetFailedAuthAttempts retrieves failed auth attempts for rate limiting
func (db *DB) GetFailedAuthAttempts(sourceIP string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM auth_audit_log
		WHERE source_ip = ? AND success = FALSE AND timestamp > ?
	`, sourceIP, since).Scan(&count)
	return count, err
}

// GetFailedAuthAttemptsForApp retrieves failed auth attempts for a specific app
func (db *DB) GetFailedAuthAttemptsForApp(appID, sourceIP string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM auth_audit_log
		WHERE app_id = ? AND source_ip = ? AND success = FALSE AND timestamp > ?
	`, appID, sourceIP, since).Scan(&count)
	return count, err
}

func scanAuditEvents(rows *sql.Rows) ([]*AuditEvent, error) {
	events := []*AuditEvent{}
	for rows.Next() {
		event := &AuditEvent{}
		var orgID, appID, failureReason, userIdentity, keyID sql.NullString

		err := rows.Scan(
			&event.ID, &event.Timestamp, &orgID, &appID, &event.AuthType, &event.Success,
			&failureReason, &event.SourceIP, &userIdentity, &keyID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}

		if orgID.Valid {
			event.OrgID = &orgID.String
		}
		if appID.Valid {
			event.AppID = &appID.String
		}
		if failureReason.Valid {
			event.FailureReason = failureReason.String
		}
		if userIdentity.Valid {
			event.UserIdentity = userIdentity.String
		}
		if keyID.Valid {
			event.KeyID = keyID.String
		}

		events = append(events, event)
	}
	return events, rows.Err()
}

// DeleteOldAuditEvents removes audit events older than the specified duration
func (db *DB) DeleteOldAuditEvents(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := db.conn.Exec(`
		DELETE FROM auth_audit_log WHERE timestamp < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountAuditEvents returns the total number of audit events
func (db *DB) CountAuditEvents() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM auth_audit_log`).Scan(&count)
	return count, err
}

// CountFailedAuthToday returns the number of failed auth attempts today
func (db *DB) CountFailedAuthToday() (int, error) {
	var count int
	today := time.Now().Truncate(24 * time.Hour)
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM auth_audit_log
		WHERE success = FALSE AND timestamp > ?
	`, today).Scan(&count)
	return count, err
}

// GetAuthStats returns authentication statistics
type AuthStats struct {
	TotalAttempts int `json:"totalAttempts"`
	SuccessCount  int `json:"successCount"`
	FailureCount  int `json:"failureCount"`
	UniqueIPs     int `json:"uniqueIps"`
	FailuresToday int `json:"failuresToday"`
}

func (db *DB) GetAuthStats() (*AuthStats, error) {
	stats := &AuthStats{}

	err := db.conn.QueryRow(`SELECT COUNT(*) FROM auth_audit_log`).Scan(&stats.TotalAttempts)
	if err != nil {
		return nil, err
	}

	err = db.conn.QueryRow(`SELECT COUNT(*) FROM auth_audit_log WHERE success = TRUE`).Scan(&stats.SuccessCount)
	if err != nil {
		return nil, err
	}

	err = db.conn.QueryRow(`SELECT COUNT(*) FROM auth_audit_log WHERE success = FALSE`).Scan(&stats.FailureCount)
	if err != nil {
		return nil, err
	}

	err = db.conn.QueryRow(`SELECT COUNT(DISTINCT source_ip) FROM auth_audit_log`).Scan(&stats.UniqueIPs)
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM auth_audit_log WHERE success = FALSE AND timestamp > ?
	`, today).Scan(&stats.FailuresToday)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
