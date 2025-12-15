package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// SessionIDLength is the length of session IDs in bytes
	SessionIDLength = 32
	// DefaultSessionDuration is the default session duration
	DefaultSessionDuration = 24 * time.Hour
)

// AuthSession represents an authenticated session (for OIDC)
type AuthSession struct {
	ID         string            `json:"id"`
	AppID      *string           `json:"appId,omitempty"`
	OrgID      *string           `json:"orgId,omitempty"`
	UserEmail  string            `json:"userEmail"`
	UserClaims map[string]string `json:"userClaims,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	ExpiresAt  time.Time         `json:"expiresAt"`
}

// GenerateSessionID generates a cryptographically secure session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, SessionIDLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new auth session
func (db *DB) CreateSession(appID, orgID *string, userEmail string, userClaims map[string]string, duration time.Duration) (*AuthSession, error) {
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil, err
	}

	if duration == 0 {
		duration = DefaultSessionDuration
	}

	now := time.Now()
	session := &AuthSession{
		ID:         sessionID,
		AppID:      appID,
		OrgID:      orgID,
		UserEmail:  userEmail,
		UserClaims: userClaims,
		CreatedAt:  now,
		ExpiresAt:  now.Add(duration),
	}

	claimsJSON, _ := json.Marshal(userClaims)

	_, err = db.conn.Exec(`
		INSERT INTO auth_sessions (id, app_id, org_id, user_email, user_claims, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.AppID, session.OrgID, session.UserEmail, string(claimsJSON), session.CreatedAt, session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by its ID
func (db *DB) GetSession(sessionID string) (*AuthSession, error) {
	session := &AuthSession{}
	var appID, orgID sql.NullString
	var claimsJSON sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, app_id, org_id, user_email, user_claims, created_at, expires_at
		FROM auth_sessions WHERE id = ?
	`, sessionID).Scan(
		&session.ID, &appID, &orgID, &session.UserEmail, &claimsJSON,
		&session.CreatedAt, &session.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if appID.Valid {
		session.AppID = &appID.String
	}
	if orgID.Valid {
		session.OrgID = &orgID.String
	}
	if claimsJSON.Valid {
		json.Unmarshal([]byte(claimsJSON.String), &session.UserClaims)
	}

	return session, nil
}

// ValidateSession retrieves a session and validates it's not expired
func (db *DB) ValidateSession(sessionID string) (*AuthSession, error) {
	session, err := db.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	// Check if session is expired
	if session.ExpiresAt.Before(time.Now()) {
		// Delete expired session
		db.DeleteSession(sessionID)
		return nil, nil
	}

	return session, nil
}

// ValidateSessionForApp validates a session for a specific app or org
func (db *DB) ValidateSessionForApp(sessionID string, appID, orgID *string) (*AuthSession, error) {
	session, err := db.ValidateSession(sessionID)
	if err != nil || session == nil {
		return nil, err
	}

	// Check if session matches the app or org
	if appID != nil && session.AppID != nil && *session.AppID == *appID {
		return session, nil
	}
	if orgID != nil && session.OrgID != nil && *session.OrgID == *orgID {
		return session, nil
	}

	// If no app_id/org_id specified in session, it's valid for all
	if session.AppID == nil && session.OrgID == nil {
		return session, nil
	}

	return nil, nil
}

// ExtendSession extends a session's expiration time
func (db *DB) ExtendSession(sessionID string, duration time.Duration) error {
	newExpiry := time.Now().Add(duration)
	_, err := db.conn.Exec(`
		UPDATE auth_sessions SET expires_at = ? WHERE id = ?
	`, newExpiry, sessionID)
	return err
}

// DeleteSession deletes a session
func (db *DB) DeleteSession(sessionID string) error {
	_, err := db.conn.Exec(`DELETE FROM auth_sessions WHERE id = ?`, sessionID)
	return err
}

// DeleteSessionsByApp deletes all sessions for an application
func (db *DB) DeleteSessionsByApp(appID string) error {
	_, err := db.conn.Exec(`DELETE FROM auth_sessions WHERE app_id = ?`, appID)
	return err
}

// DeleteSessionsByOrg deletes all sessions for an organization
func (db *DB) DeleteSessionsByOrg(orgID string) error {
	_, err := db.conn.Exec(`DELETE FROM auth_sessions WHERE org_id = ?`, orgID)
	return err
}

// DeleteExpiredSessions removes all expired sessions
func (db *DB) DeleteExpiredSessions() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM auth_sessions WHERE expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountActiveSessions returns the number of active sessions
func (db *DB) CountActiveSessions() (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM auth_sessions WHERE expires_at > ?
	`, time.Now()).Scan(&count)
	return count, err
}

// CountSessionsByApp returns the number of active sessions for an app
func (db *DB) CountSessionsByApp(appID string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM auth_sessions WHERE app_id = ? AND expires_at > ?
	`, appID, time.Now()).Scan(&count)
	return count, err
}

// OIDCState represents an OIDC state for CSRF protection
type OIDCState struct {
	ID           string    `json:"id"`
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	PKCEVerifier string    `json:"pkceVerifier"`
	RedirectURL  string    `json:"redirectUrl"`
	AppID        *string   `json:"appId,omitempty"`
	OrgID        *string   `json:"orgId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// We'll store OIDC state in auth_sessions table with a special prefix
const oidcStatePrefix = "oidc_state:"

// CreateOIDCState creates a new OIDC state for auth flow
func (db *DB) CreateOIDCState(appID, orgID *string, redirectURL, pkceVerifier string) (*OIDCState, error) {
	stateBytes := make([]byte, 32)
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	state := &OIDCState{
		ID:           uuid.New().String(),
		State:        hex.EncodeToString(stateBytes),
		Nonce:        hex.EncodeToString(nonceBytes),
		PKCEVerifier: pkceVerifier,
		RedirectURL:  redirectURL,
		AppID:        appID,
		OrgID:        orgID,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute), // OIDC state valid for 10 minutes
	}

	// Store as JSON in user_claims field
	claimsJSON, _ := json.Marshal(map[string]string{
		"type":          "oidc_state",
		"nonce":         state.Nonce,
		"pkce_verifier": state.PKCEVerifier,
		"redirect_url":  state.RedirectURL,
	})

	_, err := db.conn.Exec(`
		INSERT INTO auth_sessions (id, app_id, org_id, user_email, user_claims, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, oidcStatePrefix+state.State, state.AppID, state.OrgID, "", string(claimsJSON), state.CreatedAt, state.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC state: %w", err)
	}

	return state, nil
}

// ValidateOIDCState validates and consumes an OIDC state
func (db *DB) ValidateOIDCState(state string) (*OIDCState, error) {
	sessionID := oidcStatePrefix + state

	var appID, orgID sql.NullString
	var claimsJSON sql.NullString
	var createdAt, expiresAt time.Time

	err := db.conn.QueryRow(`
		SELECT app_id, org_id, user_claims, created_at, expires_at
		FROM auth_sessions WHERE id = ?
	`, sessionID).Scan(&appID, &orgID, &claimsJSON, &createdAt, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC state: %w", err)
	}

	// Delete the state (one-time use)
	db.DeleteSession(sessionID)

	// Check if expired
	if expiresAt.Before(time.Now()) {
		return nil, nil
	}

	// Parse claims
	var claims map[string]string
	if claimsJSON.Valid {
		json.Unmarshal([]byte(claimsJSON.String), &claims)
	}

	if claims["type"] != "oidc_state" {
		return nil, nil
	}

	oidcState := &OIDCState{
		State:        state,
		Nonce:        claims["nonce"],
		PKCEVerifier: claims["pkce_verifier"],
		RedirectURL:  claims["redirect_url"],
		CreatedAt:    createdAt,
		ExpiresAt:    expiresAt,
	}

	if appID.Valid {
		oidcState.AppID = &appID.String
	}
	if orgID.Valid {
		oidcState.OrgID = &orgID.String
	}

	return oidcState, nil
}
