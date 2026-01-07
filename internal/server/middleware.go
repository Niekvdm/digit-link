package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/policy"
)

// contextKey is a type for context keys
type contextKey string

const (
	// ContextKeyAuthResult is the context key for the auth result
	ContextKeyAuthResult contextKey = "authResult"
	// ContextKeyAuthContext is the context key for the auth context
	ContextKeyAuthContext contextKey = "authContext"
	// ContextKeyEffectivePolicy is the context key for the effective policy
	ContextKeyEffectivePolicy contextKey = "effectivePolicy"
)

// AuthMiddleware handles authentication for tunnel traffic
type AuthMiddleware struct {
	db             *db.DB
	policyResolver *policy.Resolver
	policyLoader   *policy.Loader

	// Auth handlers
	basicHandler  AuthHandler
	apiKeyHandler AuthHandler
	oidcHandler   AuthHandler

	// Rate limiter
	rateLimiter *auth.RateLimiter

	// Per-app rate limiter cache
	appRateLimiters sync.Map // map[string]*auth.RateLimiter
	appRLConfigCache sync.Map // map[string]*appRateLimitCacheEntry

	// Configuration
	defaultDeny bool // If true, deny when policy cannot be determined
}

// appRateLimitCacheEntry caches rate limit config with expiration
type appRateLimitCacheEntry struct {
	config    *db.AppRateLimitConfig
	fetchedAt time.Time
}

// AuthHandler is the interface for authentication handlers
type AuthHandler interface {
	// Authenticate attempts to authenticate the request
	Authenticate(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult

	// Challenge sends an authentication challenge to the client
	Challenge(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext)
}

// AuthMiddlewareOption is a functional option for configuring the middleware
type AuthMiddlewareOption func(*AuthMiddleware)

// WithDefaultDeny sets whether to deny by default when policy is undetermined
func WithDefaultDeny(deny bool) AuthMiddlewareOption {
	return func(m *AuthMiddleware) {
		m.defaultDeny = deny
	}
}

// WithBasicHandler sets the Basic auth handler
func WithBasicHandler(h AuthHandler) AuthMiddlewareOption {
	return func(m *AuthMiddleware) {
		m.basicHandler = h
	}
}

// WithAPIKeyHandler sets the API key auth handler
func WithAPIKeyHandler(h AuthHandler) AuthMiddlewareOption {
	return func(m *AuthMiddleware) {
		m.apiKeyHandler = h
	}
}

// WithOIDCHandler sets the OIDC auth handler
func WithOIDCHandler(h AuthHandler) AuthMiddlewareOption {
	return func(m *AuthMiddleware) {
		m.oidcHandler = h
	}
}

// WithRateLimiter sets the rate limiter
func WithRateLimiter(rl *auth.RateLimiter) AuthMiddlewareOption {
	return func(m *AuthMiddleware) {
		m.rateLimiter = rl
	}
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(database *db.DB, opts ...AuthMiddlewareOption) *AuthMiddleware {
	resolver := policy.NewResolver(
		database,
		policy.WithDefaultDenyOnError(true),
		policy.WithSecretDecryptor(auth.DecryptTOTPSecret),
	)
	loader := policy.NewLoader(database, resolver)

	m := &AuthMiddleware{
		db:             database,
		policyResolver: resolver,
		policyLoader:   loader,
		defaultDeny:    true, // Fail closed by default
		rateLimiter:    auth.NewRateLimiter(database, auth.DefaultRateLimiterConfig()),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// AuthenticateRequest authenticates an incoming request based on the subdomain
func (m *AuthMiddleware) AuthenticateRequest(w http.ResponseWriter, r *http.Request, subdomain string) (*policy.AuthResult, *policy.AuthContext) {
	// Skip auth for CORS preflight requests (OPTIONS never carry credentials)
	if r.Method == http.MethodOptions {
		return policy.Success("cors_preflight"), &policy.AuthContext{Subdomain: subdomain}
	}

	// Skip auth for internal endpoints
	if m.isInternalEndpoint(r.URL.Path) {
		return policy.Success("internal"), &policy.AuthContext{Subdomain: subdomain}
	}

	// Resolve the effective policy for this subdomain
	effectivePolicy, authCtx, err := m.policyLoader.LoadForSubdomain(subdomain)
	if err != nil {
		log.Printf("Failed to load policy for subdomain %s: %v", subdomain, err)
		if m.defaultDeny {
			return policy.Failure("failed to load auth policy"), authCtx
		}
		// Allow through if not in strict mode
		return policy.Success("policy_error_bypass"), authCtx
	}

	// If no policy is configured, allow through
	if effectivePolicy == nil || effectivePolicy.IsNone() {
		return policy.Success("no_auth_required"), authCtx
	}

	// Store policy in context
	ctx := context.WithValue(r.Context(), ContextKeyEffectivePolicy, effectivePolicy)
	ctx = context.WithValue(ctx, ContextKeyAuthContext, authCtx)
	*r = *r.WithContext(ctx)

	// Dispatch to appropriate handler
	return m.authenticate(w, r, effectivePolicy, authCtx)
}

// AuthenticateWithContext authenticates using a pre-resolved context
// This is used when we already know the org/app context (e.g., from tunnel registration)
func (m *AuthMiddleware) AuthenticateWithContext(w http.ResponseWriter, r *http.Request, authCtx *policy.AuthContext) (*policy.AuthResult, *policy.AuthContext) {
	// Skip auth for CORS preflight requests (OPTIONS never carry credentials)
	if r.Method == http.MethodOptions {
		return policy.Success("cors_preflight"), authCtx
	}

	// Skip auth for internal endpoints
	if m.isInternalEndpoint(r.URL.Path) {
		return policy.Success("internal"), authCtx
	}

	// Resolve the effective policy for this context
	effectivePolicy, err := m.policyResolver.ResolveForContext(authCtx)
	if err != nil {
		log.Printf("Failed to resolve policy for context: %v", err)
		if m.defaultDeny {
			return policy.Failure("failed to resolve auth policy"), authCtx
		}
		return policy.Success("policy_error_bypass"), authCtx
	}

	// If no policy is configured, allow through
	if effectivePolicy == nil || effectivePolicy.IsNone() {
		return policy.Success("no_auth_required"), authCtx
	}

	// Store policy in context
	ctx := context.WithValue(r.Context(), ContextKeyEffectivePolicy, effectivePolicy)
	ctx = context.WithValue(ctx, ContextKeyAuthContext, authCtx)
	*r = *r.WithContext(ctx)

	// Dispatch to appropriate handler
	return m.authenticate(w, r, effectivePolicy, authCtx)
}

// authenticate dispatches to the appropriate auth handler
func (m *AuthMiddleware) authenticate(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) (*policy.AuthResult, *policy.AuthContext) {
	// Check rate limit before authentication
	clientIP := auth.GetClientIP(r)
	rateLimitKey := auth.IPRateLimitKey(clientIP)
	if ctx != nil && ctx.AppID != "" {
		rateLimitKey = auth.AppIPRateLimitKey(ctx.AppID, clientIP)
	}

	// Get the appropriate rate limiter for this app
	rl, skipRateLimiting := m.getAppRateLimiter(ctx)

	if !skipRateLimiting && rl != nil {
		allowed, retryAfter := rl.Allow(rateLimitKey)
		if !allowed {
			// Log rate limit hit
			if m.db != nil && ctx != nil {
				var orgID, appID *string
				if ctx.OrgID != "" {
					orgID = &ctx.OrgID
				}
				if ctx.AppID != "" {
					appID = &ctx.AppID
				}
				m.db.LogAuthFailure(orgID, appID, string(p.Type), clientIP, "rate_limited")
			}

			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			return policy.Failure(fmt.Sprintf("rate limited, retry after %v", retryAfter)), ctx
		}
	}

	var result *policy.AuthResult

	// If API key is enabled as add-on, try it first
	if p.HasAPIKeyAddOn() && m.hasAPIKeyHeader(r) {
		result = m.defaultAPIKeyAuth(w, r, p, ctx)
		if result.Authenticated {
			// API key auth succeeded
			if !skipRateLimiting && rl != nil {
				rl.RecordSuccess(rateLimitKey)
			}
			return result, ctx
		}
		// API key was present but invalid - deny (don't fall back to avoid credential probing)
		if !skipRateLimiting && rl != nil {
			rl.RecordFailure(rateLimitKey)
		}
		return result, ctx
	}

	switch p.Type {
	case policy.AuthTypeBasic:
		if m.basicHandler == nil {
			result = m.defaultBasicAuth(w, r, p, ctx)
		} else {
			result = m.basicHandler.Authenticate(w, r, p, ctx)
		}

	case policy.AuthTypeAPIKey:
		if m.apiKeyHandler == nil {
			result = m.defaultAPIKeyAuth(w, r, p, ctx)
		} else {
			result = m.apiKeyHandler.Authenticate(w, r, p, ctx)
		}

	case policy.AuthTypeOIDC:
		if m.oidcHandler == nil {
			result = m.defaultOIDCAuth(w, r, p, ctx)
		} else {
			result = m.oidcHandler.Authenticate(w, r, p, ctx)
		}

	default:
		// Unknown auth type - deny if in strict mode
		if m.defaultDeny {
			result = policy.Failure("unknown auth type")
		} else {
			result = policy.Success("unknown_auth_bypass")
		}
	}

	// Record success/failure for rate limiting
	if !skipRateLimiting && rl != nil {
		if result.Authenticated {
			rl.RecordSuccess(rateLimitKey)
		} else if !result.ShouldRedirect && !result.ShouldChallenge {
			// Don't count OIDC redirects or auth challenges as failures
			// Only actual failed credential submissions should count
			rl.RecordFailure(rateLimitKey)
		}
	}

	return result, ctx
}

// hasAPIKeyHeader checks if an API key header is present (without validating)
func (m *AuthMiddleware) hasAPIKeyHeader(r *http.Request) bool {
	if r.Header.Get("X-API-Key") != "" {
		return true
	}
	if r.Header.Get("X-Tunnel-API-Key") != "" {
		return true
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return strings.HasPrefix(token, "dlk_")
	}
	return false
}

// HandleAuthResult handles the authentication result, sending appropriate responses
func (m *AuthMiddleware) HandleAuthResult(w http.ResponseWriter, r *http.Request, result *policy.AuthResult, p *policy.EffectivePolicy, ctx *policy.AuthContext) bool {
	if result.Authenticated {
		// Store result in context for downstream handlers
		newCtx := context.WithValue(r.Context(), ContextKeyAuthResult, result)
		*r = *r.WithContext(newCtx)
		return true
	}

	// Handle failed authentication
	if result.ShouldRedirect && result.RedirectURL != "" {
		http.Redirect(w, r, result.RedirectURL, http.StatusFound)
		return false
	}

	if result.ShouldChallenge && p != nil {
		m.sendChallenge(w, r, p, ctx)
		return false
	}

	// Generic 401/403 response
	if p != nil && p.Type == policy.AuthTypeBasic {
		m.sendBasicChallenge(w, ctx)
	} else {
		http.Error(w, "Unauthorized: "+result.Error, http.StatusUnauthorized)
	}
	return false
}

// sendChallenge sends an authentication challenge
func (m *AuthMiddleware) sendChallenge(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) {
	switch p.Type {
	case policy.AuthTypeBasic:
		if m.basicHandler != nil {
			m.basicHandler.Challenge(w, r, p, ctx)
		} else {
			m.sendBasicChallenge(w, ctx)
		}

	case policy.AuthTypeAPIKey:
		if m.apiKeyHandler != nil {
			m.apiKeyHandler.Challenge(w, r, p, ctx)
		} else {
			http.Error(w, "API key required", http.StatusUnauthorized)
		}

	case policy.AuthTypeOIDC:
		if m.oidcHandler != nil {
			m.oidcHandler.Challenge(w, r, p, ctx)
		} else {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
		}

	default:
		http.Error(w, "Authentication required", http.StatusUnauthorized)
	}
}

// sendBasicChallenge sends a Basic auth challenge
func (m *AuthMiddleware) sendBasicChallenge(w http.ResponseWriter, ctx *policy.AuthContext) {
	realm := "digit-link"
	if ctx != nil && ctx.Subdomain != "" {
		realm = ctx.Subdomain + ".digit-link"
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// Default auth implementations (stubs that deny by default)

func (m *AuthMiddleware) defaultBasicAuth(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult {
	// First, check for existing session cookie (avoids re-prompting on concurrent requests)
	if cookie, err := r.Cookie("digit_link_basic_session"); err == nil && cookie.Value != "" {
		if m.db != nil {
			var appID, orgID *string
			if ctx != nil {
				if ctx.AppID != "" {
					appID = &ctx.AppID
				}
				if ctx.OrgID != "" {
					orgID = &ctx.OrgID
				}
			}
			session, err := m.db.ValidateSessionForApp(cookie.Value, appID, orgID)
			if err == nil && session != nil {
				// Valid session - allow through without re-authenticating
				return policy.SuccessWithSession(session.ID, session.UserEmail)
			}
		}
	}

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
	if p.Basic == nil {
		return policy.Failure("basic auth not configured")
	}

	// Use constant-time comparison via bcrypt verify
	if !auth.VerifyPassword(password, p.Basic.PassHash) {
		return policy.Challenge("invalid credentials")
	}

	// Optionally verify username if configured
	if p.Basic.UserHash != "" && !auth.VerifyPassword(username, p.Basic.UserHash) {
		return policy.Challenge("invalid credentials")
	}

	// Create session for successful Basic Auth to avoid re-prompting on concurrent requests
	if m.db != nil {
		var appID, orgID *string
		if ctx != nil {
			if ctx.AppID != "" {
				appID = &ctx.AppID
			}
			if ctx.OrgID != "" {
				orgID = &ctx.OrgID
			}
		}
		// Determine session duration (default 24 hours if not configured)
		sessionDuration := 24 * time.Hour
		if p.Basic != nil && p.Basic.SessionDuration > 0 {
			sessionDuration = p.Basic.SessionDuration
		}
		// Create session with configured duration
		session, err := m.db.CreateSession(appID, orgID, username, map[string]string{"auth_type": "basic"}, sessionDuration)
		if err == nil && session != nil {
			// Set session cookie with matching duration
			http.SetCookie(w, &http.Cookie{
				Name:     "digit_link_basic_session",
				Value:    session.ID,
				Path:     "/",
				MaxAge:   int(sessionDuration.Seconds()),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}

	return policy.Success(username)
}

func (m *AuthMiddleware) defaultAPIKeyAuth(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult {
	// Check for API key in header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		// Try X-Tunnel-API-Key header (alias for tunnel clients)
		apiKey = r.Header.Get("X-Tunnel-API-Key")
	}
	if apiKey == "" {
		// Try Authorization: Bearer
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if apiKey == "" {
		return policy.Challenge("API key required")
	}

	// Validate API key
	key, err := m.db.ValidateAPIKey(apiKey)
	if err != nil {
		log.Printf("API key validation error: %v", err)
		return policy.Failure("API key validation error")
	}
	if key == nil {
		return policy.Failure("invalid API key")
	}

	// Check if key is for this org/app
	if ctx != nil {
		if ctx.AppID != "" && key.AppID != nil && *key.AppID != ctx.AppID {
			// Key is for a different app
			if key.OrgID == nil || *key.OrgID != ctx.OrgID {
				return policy.Failure("API key not valid for this application")
			}
		}
		if ctx.OrgID != "" && key.OrgID != nil && *key.OrgID != ctx.OrgID {
			return policy.Failure("API key not valid for this organization")
		}
	}

	// Update last used
	m.db.UpdateAPIKeyLastUsed(key.ID)

	return policy.SuccessWithKey(key.ID, key.KeyPrefix)
}

func (m *AuthMiddleware) defaultOIDCAuth(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult {
	// Check for session cookie
	cookie, err := r.Cookie("digit_link_session")
	if err != nil || cookie.Value == "" {
		// No session - need to redirect to login
		return policy.Redirect("/__auth/login?redirect=" + r.URL.RequestURI())
	}

	// Validate session
	var appID, orgID *string
	if ctx != nil {
		if ctx.AppID != "" {
			appID = &ctx.AppID
		}
		if ctx.OrgID != "" {
			orgID = &ctx.OrgID
		}
	}

	session, err := m.db.ValidateSessionForApp(cookie.Value, appID, orgID)
	if err != nil {
		log.Printf("Session validation error: %v", err)
		return policy.Redirect("/__auth/login?redirect=" + r.URL.RequestURI())
	}
	if session == nil {
		return policy.Redirect("/__auth/login?redirect=" + r.URL.RequestURI())
	}

	return policy.SuccessWithSession(session.ID, session.UserEmail)
}

// isInternalEndpoint checks if the path is an internal endpoint that should bypass auth
func (m *AuthMiddleware) isInternalEndpoint(path string) bool {
	internalPaths := []string{
		"/__auth/", // Auth endpoints
		"/health",  // Health check
		"/_tunnel", // Tunnel WebSocket
		"/setup/",  // Setup endpoints
	}

	for _, prefix := range internalPaths {
		if strings.HasPrefix(path, prefix) || path == strings.TrimSuffix(prefix, "/") {
			return true
		}
	}

	return false
}

// GetAuthResultFromContext retrieves the auth result from request context
func GetAuthResultFromContext(r *http.Request) *policy.AuthResult {
	if result, ok := r.Context().Value(ContextKeyAuthResult).(*policy.AuthResult); ok {
		return result
	}
	return nil
}

// GetAuthContextFromContext retrieves the auth context from request context
func GetAuthContextFromContext(r *http.Request) *policy.AuthContext {
	if ctx, ok := r.Context().Value(ContextKeyAuthContext).(*policy.AuthContext); ok {
		return ctx
	}
	return nil
}

// GetEffectivePolicyFromContext retrieves the effective policy from request context
func GetEffectivePolicyFromContext(r *http.Request) *policy.EffectivePolicy {
	if p, ok := r.Context().Value(ContextKeyEffectivePolicy).(*policy.EffectivePolicy); ok {
		return p
	}
	return nil
}

// InvalidateSubdomainCache invalidates the cached policy for a subdomain
func (m *AuthMiddleware) InvalidateSubdomainCache(subdomain string) {
	if m.policyLoader != nil {
		m.policyLoader.InvalidateSubdomain(subdomain)
	}
}

// InvalidateAppCache invalidates the cached policy for an app
func (m *AuthMiddleware) InvalidateAppCache(appID string) {
	if m.policyLoader != nil {
		m.policyLoader.InvalidateApp(appID)
	}
}

// InvalidateOrgCache invalidates the cached policy for an org
func (m *AuthMiddleware) InvalidateOrgCache(orgID string) {
	if m.policyLoader != nil {
		m.policyLoader.InvalidateOrg(orgID)
	}
}

// getAppRateLimiter returns the appropriate rate limiter for an app
// Returns (rateLimiter, skipRateLimiting) where skipRateLimiting=true means rate limiting is disabled
func (m *AuthMiddleware) getAppRateLimiter(ctx *policy.AuthContext) (*auth.RateLimiter, bool) {
	// If no app context, use default rate limiter
	if ctx == nil || ctx.AppID == "" {
		return m.rateLimiter, false
	}

	// Check cache first (with 5-minute TTL)
	const cacheTTL = 5 * time.Minute
	if cached, ok := m.appRLConfigCache.Load(ctx.AppID); ok {
		entry := cached.(*appRateLimitCacheEntry)
		if time.Since(entry.fetchedAt) < cacheTTL {
			if entry.config == nil {
				// No custom config, use default
				return m.rateLimiter, false
			}
			if !entry.config.Enabled {
				// Rate limiting disabled for this app
				return nil, true
			}
			// Use cached custom rate limiter
			if rl, ok := m.appRateLimiters.Load(ctx.AppID); ok {
				return rl.(*auth.RateLimiter), false
			}
		}
	}

	// Fetch from database
	config, err := m.db.GetAppRateLimitConfig(ctx.AppID)
	if err != nil {
		log.Printf("Failed to get rate limit config for app %s: %v", ctx.AppID, err)
		// Fall back to default on error
		return m.rateLimiter, false
	}

	// Cache the config
	m.appRLConfigCache.Store(ctx.AppID, &appRateLimitCacheEntry{
		config:    config,
		fetchedAt: time.Now(),
	})

	// No custom config, use default
	if config == nil {
		return m.rateLimiter, false
	}

	// Rate limiting disabled for this app
	if !config.Enabled {
		return nil, true
	}

	// Create or get custom rate limiter for this app
	if rl, ok := m.appRateLimiters.Load(ctx.AppID); ok {
		return rl.(*auth.RateLimiter), false
	}

	// Create new rate limiter with custom config
	customConfig := auth.RateLimiterConfig{
		MaxAttempts:     config.MaxAttempts,
		WindowDuration:  time.Duration(config.WindowDurationSeconds) * time.Second,
		BlockDuration:   time.Duration(config.BlockDurationSeconds) * time.Second,
		CleanupInterval: 5 * time.Minute,
	}
	customRL := auth.NewRateLimiter(m.db, customConfig)
	m.appRateLimiters.Store(ctx.AppID, customRL)

	return customRL, false
}

// InvalidateAppRateLimitCache invalidates the cached rate limit config for an app
func (m *AuthMiddleware) InvalidateAppRateLimitCache(appID string) {
	m.appRLConfigCache.Delete(appID)
	// Also remove the custom rate limiter so it gets recreated with new config
	if rl, ok := m.appRateLimiters.LoadAndDelete(appID); ok {
		rl.(*auth.RateLimiter).Stop()
	}
}
