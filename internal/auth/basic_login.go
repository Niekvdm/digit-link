package auth

import (
	"html/template"
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

// LoginPageData contains data for rendering the login page
type LoginPageData struct {
	Subdomain string
	Realm     string
	ReturnURL string
	LoginURL  string
	Username  string
	Error     string
}

// BasicAuthLoginHandler handles the Basic Auth login flow
// It serves a custom login page and handles form submissions
type BasicAuthLoginHandler struct {
	db       *db.DB
	scheme   string // "http" or "https" for cookie security
	template *template.Template
}

// NewBasicAuthLoginHandler creates a new BasicAuthLoginHandler
func NewBasicAuthLoginHandler(database *db.DB, scheme string) *BasicAuthLoginHandler {
	tmpl := template.Must(template.New("login").Parse(BasicLoginTemplate))
	return &BasicAuthLoginHandler{
		db:       database,
		scheme:   scheme,
		template: tmpl,
	}
}

// LoginConfig contains configuration for a login attempt
type LoginConfig struct {
	Policy    *policy.EffectivePolicy
	AuthCtx   *policy.AuthContext
	ReturnURL string
}

// HandleLogin handles the login endpoint
// GET: Renders the login page
// POST: Handles form submission
func (h *BasicAuthLoginHandler) HandleLogin(w http.ResponseWriter, r *http.Request, config *LoginConfig) {
	if config == nil || config.Policy == nil || config.Policy.Basic == nil {
		http.Error(w, "Basic auth not configured", http.StatusInternalServerError)
		return
	}

	subdomain := ""
	if config.AuthCtx != nil {
		subdomain = config.AuthCtx.Subdomain
	}

	switch r.Method {
	case http.MethodGet:
		h.renderLoginPage(w, subdomain, config.ReturnURL, "", "")
	case http.MethodPost:
		h.handleFormSubmit(w, r, config)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// renderLoginPage renders the login HTML page
func (h *BasicAuthLoginHandler) renderLoginPage(w http.ResponseWriter, subdomain, returnURL, username, errorMsg string) {
	realm := "digit-link"
	if subdomain != "" {
		realm = subdomain + ".digit-link"
	}

	data := LoginPageData{
		Subdomain: subdomain,
		Realm:     realm,
		ReturnURL: returnURL,
		LoginURL:  BasicAuthLoginPath,
		Username:  username,
		Error:     errorMsg,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	if err := h.template.Execute(w, data); err != nil {
		http.Error(w, "Failed to render login page", http.StatusInternalServerError)
	}
}

// handleFormSubmit handles the login form POST submission
func (h *BasicAuthLoginHandler) handleFormSubmit(w http.ResponseWriter, r *http.Request, config *LoginConfig) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.renderLoginPage(w, config.AuthCtx.Subdomain, config.ReturnURL, "", "Invalid form data")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	returnURL := r.FormValue("return")
	subdomain := r.FormValue("subdomain")

	if returnURL == "" {
		returnURL = config.ReturnURL
	}
	if subdomain == "" && config.AuthCtx != nil {
		subdomain = config.AuthCtx.Subdomain
	}

	// Validate required fields
	if username == "" || password == "" {
		h.renderLoginPage(w, subdomain, returnURL, username, "Username and password are required")
		return
	}

	// Validate password
	if !VerifyPassword(password, config.Policy.Basic.PassHash) {
		h.logFailure(config.AuthCtx, r, "invalid_password")
		h.renderLoginPage(w, subdomain, returnURL, username, "Invalid username or password")
		return
	}

	// Optionally verify username if configured
	if config.Policy.Basic.UserHash != "" {
		if !VerifyPassword(username, config.Policy.Basic.UserHash) {
			h.logFailure(config.AuthCtx, r, "invalid_username")
			h.renderLoginPage(w, subdomain, returnURL, username, "Invalid username or password")
			return
		}
	}

	// Credentials valid - create session
	sessionID, err := h.createSession(config.AuthCtx, username, config.Policy.Basic.SessionDuration)
	if err != nil {
		h.renderLoginPage(w, subdomain, returnURL, username, "Failed to create session")
		return
	}

	// Set session cookie
	h.setSessionCookie(w, sessionID, config.Policy.Basic.SessionDuration)

	// Log success
	h.logSuccess(config.AuthCtx, r, username)

	// Redirect back to original URL
	if returnURL == "" {
		returnURL = "/"
	}
	http.Redirect(w, r, returnURL, http.StatusFound)
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
