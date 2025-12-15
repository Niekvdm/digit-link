package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/policy"
	"golang.org/x/oauth2"
)

// OIDCAuthHandler handles OIDC/OAuth2 authentication
type OIDCAuthHandler struct {
	db            *db.DB
	domain        string
	sessionCookie string

	// Provider cache
	providers   map[string]*cachedOIDCProvider
	providersMu sync.RWMutex
}

type cachedOIDCProvider struct {
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	createdAt    time.Time
}

// NewOIDCAuthHandler creates a new OIDC auth handler
func NewOIDCAuthHandler(database *db.DB, domain string) *OIDCAuthHandler {
	return &OIDCAuthHandler{
		db:            database,
		domain:        domain,
		sessionCookie: "digit_link_session",
		providers:     make(map[string]*cachedOIDCProvider),
	}
}

// Authenticate implements the AuthHandler interface for OIDC auth
func (h *OIDCAuthHandler) Authenticate(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult {
	// Check for existing session
	cookie, err := r.Cookie(h.sessionCookie)
	if err == nil && cookie.Value != "" {
		session, err := h.validateSession(cookie.Value, ctx)
		if err == nil && session != nil {
			return policy.SuccessWithSession(session.ID, session.UserEmail)
		}
	}

	// No valid session - need to redirect to login
	redirectURL := r.URL.RequestURI()
	loginURL := fmt.Sprintf("/__auth/login?redirect=%s", url.QueryEscape(redirectURL))

	return policy.Redirect(loginURL)
}

// Challenge sends an OIDC auth challenge (redirect to login)
func (h *OIDCAuthHandler) Challenge(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) {
	redirectURL := r.URL.RequestURI()
	loginURL := fmt.Sprintf("/__auth/login?redirect=%s", url.QueryEscape(redirectURL))
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// HandleLogin handles the login endpoint - starts OIDC flow
func (h *OIDCAuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) {
	if p == nil || p.OIDC == nil {
		http.Error(w, "OIDC not configured", http.StatusInternalServerError)
		return
	}

	// Get redirect URL from query param
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Get or create OIDC provider
	provider, err := h.getOrCreateProvider(r.Context(), p.OIDC)
	if err != nil {
		log.Printf("Failed to create OIDC provider: %v", err)
		http.Error(w, "Failed to initialize OIDC provider", http.StatusInternalServerError)
		return
	}

	// Generate PKCE verifier and challenge
	verifier, challenge, err := generatePKCE()
	if err != nil {
		log.Printf("Failed to generate PKCE: %v", err)
		http.Error(w, "Failed to initialize authentication", http.StatusInternalServerError)
		return
	}

	// Create state in database
	var appID, orgID *string
	if ctx != nil {
		if ctx.AppID != "" {
			appID = &ctx.AppID
		}
		if ctx.OrgID != "" {
			orgID = &ctx.OrgID
		}
	}

	state, err := h.db.CreateOIDCState(appID, orgID, redirectURL, verifier)
	if err != nil {
		log.Printf("Failed to create OIDC state: %v", err)
		http.Error(w, "Failed to initialize authentication", http.StatusInternalServerError)
		return
	}

	// Build authorization URL with PKCE
	authURL := provider.oauth2Config.AuthCodeURL(
		state.State,
		oauth2.SetAuthURLParam("nonce", state.Nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback handles the OIDC callback endpoint
func (h *OIDCAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) {
	if p == nil || p.OIDC == nil {
		http.Error(w, "OIDC not configured", http.StatusInternalServerError)
		return
	}

	// Get authorization code and state from query params
	code := r.URL.Query().Get("code")
	stateParam := r.URL.Query().Get("state")

	if code == "" || stateParam == "" {
		// Check for error
		errorParam := r.URL.Query().Get("error")
		errorDesc := r.URL.Query().Get("error_description")
		if errorParam != "" {
			log.Printf("OIDC error: %s - %s", errorParam, errorDesc)
			http.Error(w, fmt.Sprintf("Authentication failed: %s", errorDesc), http.StatusUnauthorized)
			return
		}
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}

	// Validate and consume state
	state, err := h.db.ValidateOIDCState(stateParam)
	if err != nil {
		log.Printf("Failed to validate OIDC state: %v", err)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	if state == nil {
		http.Error(w, "Invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	// Get provider
	provider, err := h.getOrCreateProvider(r.Context(), p.OIDC)
	if err != nil {
		log.Printf("Failed to get OIDC provider: %v", err)
		http.Error(w, "Failed to validate authentication", http.StatusInternalServerError)
		return
	}

	// Exchange code for token with PKCE verifier
	token, err := provider.oauth2Config.Exchange(
		r.Context(),
		code,
		oauth2.SetAuthURLParam("code_verifier", state.PKCEVerifier),
	)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Printf("No id_token in token response")
		http.Error(w, "Missing ID token", http.StatusInternalServerError)
		return
	}

	idToken, err := provider.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("Failed to verify ID token: %v", err)
		http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
		return
	}

	// Verify nonce
	var claims struct {
		Nonce         string `json:"nonce"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Subject       string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("Failed to parse claims: %v", err)
		http.Error(w, "Failed to parse ID token claims", http.StatusInternalServerError)
		return
	}

	if claims.Nonce != state.Nonce {
		log.Printf("Nonce mismatch")
		http.Error(w, "Invalid nonce", http.StatusUnauthorized)
		return
	}

	// Validate claims against policy
	if err := h.validateClaims(&claims, p.OIDC); err != nil {
		log.Printf("Claims validation failed: %v", err)

		// Log failed auth
		if state.OrgID != nil || state.AppID != nil {
			h.db.LogAuthFailure(state.OrgID, state.AppID, "oidc", GetClientIPFromRequest(r), err.Error())
		}

		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Create session
	userClaims := map[string]string{
		"sub":   claims.Subject,
		"email": claims.Email,
		"name":  claims.Name,
	}

	session, err := h.db.CreateSession(state.AppID, state.OrgID, claims.Email, userClaims, 24*time.Hour)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Log successful auth
	if state.OrgID != nil || state.AppID != nil {
		h.db.LogAuthSuccess(state.OrgID, state.AppID, "oidc", GetClientIPFromRequest(r), claims.Email, "")
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionCookie,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to original URL
	http.Redirect(w, r, state.RedirectURL, http.StatusFound)
}

// HandleLogout handles the logout endpoint
func (h *OIDCAuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie
	cookie, err := r.Cookie(h.sessionCookie)
	if err == nil && cookie.Value != "" {
		// Delete session from database
		h.db.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to home or specified URL
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// validateSession validates a session ID and returns the session if valid
func (h *OIDCAuthHandler) validateSession(sessionID string, ctx *policy.AuthContext) (*db.AuthSession, error) {
	var appID, orgID *string
	if ctx != nil {
		if ctx.AppID != "" {
			appID = &ctx.AppID
		}
		if ctx.OrgID != "" {
			orgID = &ctx.OrgID
		}
	}

	return h.db.ValidateSessionForApp(sessionID, appID, orgID)
}

// validateClaims validates ID token claims against policy requirements
func (h *OIDCAuthHandler) validateClaims(claims *struct {
	Nonce         string `json:"nonce"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Subject       string `json:"sub"`
}, config *policy.OIDCConfig) error {
	// Check email domain restriction
	if len(config.AllowedDomains) > 0 {
		if claims.Email == "" {
			return fmt.Errorf("email claim required but not provided")
		}

		parts := strings.Split(claims.Email, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid email format")
		}

		domain := strings.ToLower(parts[1])
		allowed := false
		for _, d := range config.AllowedDomains {
			if strings.ToLower(d) == domain {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("email domain '%s' not allowed", domain)
		}
	}

	// Note: Required claims validation would need access to the full claims map
	// For advanced claim validation, use ValidateClaimsExtended

	return nil
}

// ValidateClaimsExtended validates claims with full access to all claim values
// This is used when you need to validate arbitrary claims beyond email domain
func (h *OIDCAuthHandler) ValidateClaimsExtended(claims map[string]interface{}, config *policy.OIDCConfig) error {
	// Check email domain restriction
	if len(config.AllowedDomains) > 0 {
		email, ok := claims["email"].(string)
		if !ok || email == "" {
			return fmt.Errorf("email claim required but not provided")
		}

		parts := strings.Split(email, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid email format")
		}

		domain := strings.ToLower(parts[1])
		allowed := false
		for _, d := range config.AllowedDomains {
			if strings.ToLower(d) == domain {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("email domain '%s' not allowed", domain)
		}
	}

	// Check required claims
	for requiredKey, requiredValue := range config.RequiredClaims {
		actualValue, ok := claims[requiredKey]
		if !ok {
			return fmt.Errorf("required claim '%s' not present", requiredKey)
		}

		// Handle different value types
		switch v := actualValue.(type) {
		case string:
			if v != requiredValue {
				return fmt.Errorf("claim '%s' value '%s' does not match required value '%s'", requiredKey, v, requiredValue)
			}
		case []interface{}:
			// Check if required value is in array (e.g., groups claim)
			found := false
			for _, item := range v {
				if itemStr, ok := item.(string); ok && itemStr == requiredValue {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("claim '%s' does not contain required value '%s'", requiredKey, requiredValue)
			}
		case bool:
			if requiredValue == "true" && !v {
				return fmt.Errorf("claim '%s' must be true", requiredKey)
			}
			if requiredValue == "false" && v {
				return fmt.Errorf("claim '%s' must be false", requiredKey)
			}
		default:
			// For other types, do string comparison
			if fmt.Sprintf("%v", v) != requiredValue {
				return fmt.Errorf("claim '%s' value does not match required value '%s'", requiredKey, requiredValue)
			}
		}
	}

	return nil
}

// getOrCreateProvider gets or creates an OIDC provider for the given config
func (h *OIDCAuthHandler) getOrCreateProvider(ctx context.Context, config *policy.OIDCConfig) (*cachedOIDCProvider, error) {
	h.providersMu.RLock()
	provider, ok := h.providers[config.IssuerURL]
	h.providersMu.RUnlock()

	if ok && time.Since(provider.createdAt) < 24*time.Hour {
		return provider, nil
	}

	// Create new provider
	h.providersMu.Lock()
	defer h.providersMu.Unlock()

	// Double-check after acquiring write lock
	provider, ok = h.providers[config.IssuerURL]
	if ok && time.Since(provider.createdAt) < 24*time.Hour {
		return provider, nil
	}

	oidcProv, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	// Build redirect URL (will be set per-request based on subdomain)
	redirectURL := fmt.Sprintf("https://%s/__auth/callback", h.domain)

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     oidcProv.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	verifier := oidcProv.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	provider = &cachedOIDCProvider{
		provider:     oidcProv,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		createdAt:    time.Now(),
	}

	h.providers[config.IssuerURL] = provider
	return provider, nil
}

// generatePKCE generates a PKCE code verifier and challenge
func generatePKCE() (verifier, challenge string, err error) {
	// Generate 32 random bytes for verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", err
	}

	verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Create S256 challenge
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])

	return verifier, challenge, nil
}

// GetRedirectURLForSubdomain returns the callback URL for a specific subdomain
func (h *OIDCAuthHandler) GetRedirectURLForSubdomain(subdomain string) string {
	return fmt.Sprintf("https://%s.%s/__auth/callback", subdomain, h.domain)
}

// UpdateProviderRedirectURL updates the OAuth2 config redirect URL for a specific subdomain
func (h *OIDCAuthHandler) UpdateProviderRedirectURL(issuerURL, subdomain string) {
	h.providersMu.Lock()
	defer h.providersMu.Unlock()

	if provider, ok := h.providers[issuerURL]; ok {
		provider.oauth2Config.RedirectURL = h.GetRedirectURLForSubdomain(subdomain)
	}
}
