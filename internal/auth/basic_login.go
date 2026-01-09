package auth

import (
	"net/http"
	"net/url"
	"time"

	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/policy"
)

const (
	// BasicAuthLoginPath is the path for the basic auth login endpoint
	BasicAuthLoginPath = "/_auth/basic/login"

	// BasicAuthSessionCookie is the name of the session cookie
	BasicAuthSessionCookie = "digit_link_basic_session"

	// DefaultBasicSessionDuration is the default session duration for basic auth
	DefaultBasicSessionDuration = 24 * time.Hour
)

// BasicAuthLoginHandler handles the Basic Auth login flow
// It provides a single endpoint that triggers the browser's Basic Auth prompt,
// validates credentials, creates a session, and redirects back to the original URL.
type BasicAuthLoginHandler struct {
	db     *db.DB
	scheme string // "http" or "https" for cookie security
}

// NewBasicAuthLoginHandler creates a new BasicAuthLoginHandler
func NewBasicAuthLoginHandler(database *db.DB, scheme string) *BasicAuthLoginHandler {
	return &BasicAuthLoginHandler{
		db:     database,
		scheme: scheme,
	}
}

// LoginConfig contains configuration for a login attempt
type LoginConfig struct {
	Policy    *policy.EffectivePolicy
	AuthCtx   *policy.AuthContext
	ReturnURL string
}

// HandleLogin handles the login endpoint
// GET /_auth/basic/login?return=<url>&subdomain=<subdomain>
func (h *BasicAuthLoginHandler) HandleLogin(w http.ResponseWriter, r *http.Request, config *LoginConfig) {
	if config == nil || config.Policy == nil || config.Policy.Basic == nil {
		http.Error(w, "Basic auth not configured", http.StatusInternalServerError)
		return
	}

	// Check for Authorization header
	username, password, ok := r.BasicAuth()
	if !ok {
		// No credentials - send 401 challenge
		h.sendChallenge(w, config.AuthCtx)
		return
	}

	// Validate credentials
	if !VerifyPassword(password, config.Policy.Basic.PassHash) {
		h.logFailure(config.AuthCtx, r, "invalid_password")
		h.sendChallenge(w, config.AuthCtx)
		return
	}

	// Optionally verify username if configured
	if config.Policy.Basic.UserHash != "" {
		if !VerifyPassword(username, config.Policy.Basic.UserHash) {
			h.logFailure(config.AuthCtx, r, "invalid_username")
			h.sendChallenge(w, config.AuthCtx)
			return
		}
	}

	// Credentials valid - create session
	sessionID, err := h.createSession(config.AuthCtx, username, config.Policy.Basic.SessionDuration)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	h.setSessionCookie(w, sessionID, config.Policy.Basic.SessionDuration)

	// Log success
	h.logSuccess(config.AuthCtx, r, username)

	// Redirect back to original URL
	returnURL := config.ReturnURL
	if returnURL == "" {
		returnURL = "/"
	}
	http.Redirect(w, r, returnURL, http.StatusFound)
}

// sendChallenge sends the WWW-Authenticate challenge
func (h *BasicAuthLoginHandler) sendChallenge(w http.ResponseWriter, ctx *policy.AuthContext) {
	realm := "digit-link"
	if ctx != nil && ctx.Subdomain != "" {
		realm = ctx.Subdomain + ".digit-link"
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// createSession creates a new auth session and returns the session ID
func (h *BasicAuthLoginHandler) createSession(ctx *policy.AuthContext, username string, duration time.Duration) (string, error) {
	if h.db == nil {
		return "", nil
	}

	if duration == 0 {
		duration = DefaultBasicSessionDuration
	}

	var appID, orgID *string
	if ctx != nil {
		if ctx.AppID != "" {
			appID = &ctx.AppID
		}
		if ctx.OrgID != "" {
			orgID = &ctx.OrgID
		}
	}

	session, err := h.db.CreateSession(appID, orgID, username, map[string]string{"auth_type": "basic"}, duration)
	if err != nil {
		return "", err
	}
	if session == nil {
		return "", nil
	}
	return session.ID, nil
}

// setSessionCookie sets the session cookie on the response
func (h *BasicAuthLoginHandler) setSessionCookie(w http.ResponseWriter, sessionID string, duration time.Duration) {
	if sessionID == "" {
		return
	}

	if duration == 0 {
		duration = DefaultBasicSessionDuration
	}

	http.SetCookie(w, &http.Cookie{
		Name:     BasicAuthSessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: true,
		Secure:   h.scheme == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

// logFailure logs a failed authentication attempt
func (h *BasicAuthLoginHandler) logFailure(ctx *policy.AuthContext, r *http.Request, reason string) {
	if h.db == nil || ctx == nil {
		return
	}

	var orgID, appID *string
	if ctx.OrgID != "" {
		orgID = &ctx.OrgID
	}
	if ctx.AppID != "" {
		appID = &ctx.AppID
	}
	h.db.LogAuthFailure(orgID, appID, "basic", GetClientIP(r), reason)
}

// logSuccess logs a successful authentication
func (h *BasicAuthLoginHandler) logSuccess(ctx *policy.AuthContext, r *http.Request, username string) {
	if h.db == nil || ctx == nil {
		return
	}

	var orgID, appID *string
	if ctx.OrgID != "" {
		orgID = &ctx.OrgID
	}
	if ctx.AppID != "" {
		appID = &ctx.AppID
	}
	h.db.LogAuthSuccess(orgID, appID, "basic", GetClientIP(r), username, "")
}

// ValidateSession validates a session cookie and returns the session if valid
func (h *BasicAuthLoginHandler) ValidateSession(r *http.Request, appID, orgID *string) (*db.AuthSession, error) {
	if h.db == nil {
		return nil, nil
	}

	cookie, err := r.Cookie(BasicAuthSessionCookie)
	if err != nil || cookie.Value == "" {
		return nil, nil
	}

	return h.db.ValidateSessionForApp(cookie.Value, appID, orgID)
}

// BuildLoginURL builds the login URL with return parameter
func BuildLoginURL(returnURL, subdomain string) string {
	loginURL := BasicAuthLoginPath + "?"
	params := url.Values{}
	if returnURL != "" {
		params.Set("return", returnURL)
	}
	if subdomain != "" {
		params.Set("subdomain", subdomain)
	}
	return loginURL + params.Encode()
}

// RedirectToLogin redirects the request to the login endpoint
func RedirectToLogin(w http.ResponseWriter, r *http.Request, subdomain string) {
	// Build the return URL (current request URL)
	returnURL := r.URL.String()

	// Build login URL
	loginURL := BuildLoginURL(returnURL, subdomain)

	// Redirect
	http.Redirect(w, r, loginURL, http.StatusFound)
}
