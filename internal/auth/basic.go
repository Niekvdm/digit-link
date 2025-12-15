package auth

import (
	"net/http"
	"strings"

	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/policy"
)

// BasicAuthHandler handles Basic authentication
type BasicAuthHandler struct {
	db *db.DB
}

// NewBasicAuthHandler creates a new Basic auth handler
func NewBasicAuthHandler(database *db.DB) *BasicAuthHandler {
	return &BasicAuthHandler{db: database}
}

// Authenticate implements the AuthHandler interface for Basic auth
func (h *BasicAuthHandler) Authenticate(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult {
	// Check for Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		return policy.Challenge("basic auth required")
	}

	// Parse credentials
	username, password, ok := r.BasicAuth()
	if !ok {
		return policy.Challenge("invalid basic auth header")
	}

	// Verify credentials against policy
	if p == nil || p.Basic == nil {
		return policy.Failure("basic auth not configured")
	}

	// Verify password using constant-time bcrypt comparison
	if !VerifyPassword(password, p.Basic.PassHash) {
		// Log failed attempt for audit
		if h.db != nil && ctx != nil {
			var orgID, appID *string
			if ctx.OrgID != "" {
				orgID = &ctx.OrgID
			}
			if ctx.AppID != "" {
				appID = &ctx.AppID
			}
			h.db.LogAuthFailure(orgID, appID, "basic", GetClientIPFromRequest(r), "invalid_password")
		}
		return policy.Challenge("invalid credentials")
	}

	// Optionally verify username if configured (username hash is optional)
	if p.Basic.UserHash != "" {
		if !VerifyPassword(username, p.Basic.UserHash) {
			if h.db != nil && ctx != nil {
				var orgID, appID *string
				if ctx.OrgID != "" {
					orgID = &ctx.OrgID
				}
				if ctx.AppID != "" {
					appID = &ctx.AppID
				}
				h.db.LogAuthFailure(orgID, appID, "basic", GetClientIPFromRequest(r), "invalid_username")
			}
			return policy.Challenge("invalid credentials")
		}
	}

	// Log successful authentication
	if h.db != nil && ctx != nil {
		var orgID, appID *string
		if ctx.OrgID != "" {
			orgID = &ctx.OrgID
		}
		if ctx.AppID != "" {
			appID = &ctx.AppID
		}
		h.db.LogAuthSuccess(orgID, appID, "basic", GetClientIPFromRequest(r), username, "")
	}

	return policy.Success(username)
}

// Challenge sends a Basic auth challenge to the client
func (h *BasicAuthHandler) Challenge(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) {
	realm := "digit-link"
	if ctx != nil && ctx.Subdomain != "" {
		realm = ctx.Subdomain + ".digit-link"
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// GetClientIPFromRequest extracts the client IP from a request
// This is a wrapper around GetClientIP for requests
func GetClientIPFromRequest(r *http.Request) string {
	return GetClientIP(r)
}

// SetBasicAuthCredentials creates or updates Basic auth credentials for an org policy
func SetBasicAuthCredentials(database *db.DB, orgID, username, password string) error {
	passHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	var userHash string
	if username != "" {
		userHash, err = HashPassword(username)
		if err != nil {
			return err
		}
	}

	policy := &db.OrgAuthPolicy{
		OrgID:         orgID,
		AuthType:      db.AuthTypeBasic,
		BasicUserHash: userHash,
		BasicPassHash: passHash,
	}

	return database.CreateOrgAuthPolicy(policy)
}

// SetBasicAuthCredentialsForApp creates or updates Basic auth credentials for an app policy
func SetBasicAuthCredentialsForApp(database *db.DB, appID, username, password string) error {
	passHash, err := HashPassword(password)
	if err != nil {
		return err
	}

	var userHash string
	if username != "" {
		userHash, err = HashPassword(username)
		if err != nil {
			return err
		}
	}

	policy := &db.AppAuthPolicy{
		AppID:         appID,
		AuthType:      db.AuthTypeBasic,
		BasicUserHash: userHash,
		BasicPassHash: passHash,
	}

	return database.CreateAppAuthPolicy(policy)
}

// ValidateBasicCredentials validates Basic auth credentials against stored hashes
// This is a utility function for testing/validation
func ValidateBasicCredentials(username, password, userHash, passHash string) bool {
	// Password is required
	if !VerifyPassword(password, passHash) {
		return false
	}

	// Username is optional
	if userHash != "" && !VerifyPassword(username, userHash) {
		return false
	}

	return true
}
