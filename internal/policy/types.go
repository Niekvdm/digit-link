package policy

import (
	"time"

	"github.com/niekvdm/digit-link/internal/db"
)

// AuthMode represents the authentication mode for an application
type AuthMode string

const (
	AuthModeInherit  AuthMode = "inherit"
	AuthModeDisabled AuthMode = "disabled"
	AuthModeCustom   AuthMode = "custom"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeNone   AuthType = ""
	AuthTypeBasic  AuthType = "basic"
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeOIDC   AuthType = "oidc"
)

// BasicConfig holds Basic auth configuration
type BasicConfig struct {
	UserHash        string
	PassHash        string
	SessionDuration time.Duration // 0 = use default (24h)
}

// APIKeyConfig holds API key auth configuration
type APIKeyConfig struct {
	// API keys are stored separately in the api_keys table
	// This config is just a marker that API key auth is enabled
}

// OIDCConfig holds OIDC auth configuration
type OIDCConfig struct {
	IssuerURL      string
	ClientID       string
	ClientSecret   string // Decrypted secret
	Scopes         []string
	AllowedDomains []string
	RequiredClaims map[string]string
}

// EffectivePolicy represents the resolved authentication policy
// This is what the middleware actually enforces
type EffectivePolicy struct {
	// Type is the authentication type to enforce
	Type AuthType

	// OrgID is the organization this policy belongs to
	OrgID string

	// AppID is the application this policy belongs to (empty for org-level)
	AppID string

	// Basic holds Basic auth configuration (if Type == AuthTypeBasic)
	Basic *BasicConfig

	// APIKey holds API key auth configuration (if Type == AuthTypeAPIKey)
	APIKey *APIKeyConfig

	// OIDC holds OIDC auth configuration (if Type == AuthTypeOIDC)
	OIDC *OIDCConfig
}

// IsNone returns true if no authentication is required
func (p *EffectivePolicy) IsNone() bool {
	return p == nil || p.Type == AuthTypeNone
}

// IsBasic returns true if Basic auth is required
func (p *EffectivePolicy) IsBasic() bool {
	return p != nil && p.Type == AuthTypeBasic
}

// IsAPIKey returns true if API key auth is required
func (p *EffectivePolicy) IsAPIKey() bool {
	return p != nil && p.Type == AuthTypeAPIKey
}

// IsOIDC returns true if OIDC auth is required
func (p *EffectivePolicy) IsOIDC() bool {
	return p != nil && p.Type == AuthTypeOIDC
}

// AuthContext represents the context for an authentication request
type AuthContext struct {
	// Subdomain is the subdomain being accessed
	Subdomain string

	// OrgID is the organization ID (if known)
	OrgID string

	// AppID is the application ID (if this is a persistent app)
	AppID string

	// App is the application record (if this is a persistent app)
	App *db.Application

	// IsPersistentApp is true if this is a persistent app (not random subdomain)
	IsPersistentApp bool
}

// AuthResult represents the result of an authentication attempt
type AuthResult struct {
	// Authenticated is true if authentication succeeded
	Authenticated bool

	// UserIdentity is the identity of the authenticated user (email, username, etc.)
	UserIdentity string

	// KeyID is the API key ID if authenticated via API key
	KeyID string

	// SessionID is the session ID if authenticated via OIDC
	SessionID string

	// Error is the error message if authentication failed
	Error string

	// ShouldChallenge is true if the client should be challenged (e.g., 401 for Basic)
	ShouldChallenge bool

	// ShouldRedirect is true if the client should be redirected (e.g., OIDC login)
	ShouldRedirect bool

	// RedirectURL is the URL to redirect to (if ShouldRedirect is true)
	RedirectURL string
}

// Success returns a successful auth result
func Success(userIdentity string) *AuthResult {
	return &AuthResult{
		Authenticated: true,
		UserIdentity:  userIdentity,
	}
}

// SuccessWithKey returns a successful auth result for API key auth
func SuccessWithKey(keyID, keyPrefix string) *AuthResult {
	return &AuthResult{
		Authenticated: true,
		KeyID:         keyID,
		UserIdentity:  "api_key:" + keyPrefix,
	}
}

// SuccessWithSession returns a successful auth result for OIDC auth
func SuccessWithSession(sessionID, userEmail string) *AuthResult {
	return &AuthResult{
		Authenticated: true,
		SessionID:     sessionID,
		UserIdentity:  userEmail,
	}
}

// Failure returns a failed auth result
func Failure(err string) *AuthResult {
	return &AuthResult{
		Authenticated: false,
		Error:         err,
	}
}

// Challenge returns a result indicating the client should be challenged
func Challenge(err string) *AuthResult {
	return &AuthResult{
		Authenticated:   false,
		Error:           err,
		ShouldChallenge: true,
	}
}

// Redirect returns a result indicating the client should be redirected
func Redirect(url string) *AuthResult {
	return &AuthResult{
		Authenticated:  false,
		ShouldRedirect: true,
		RedirectURL:    url,
	}
}
