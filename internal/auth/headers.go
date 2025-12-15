package auth

import (
	"net/http"
)

// SecurityHeaders adds security-related HTTP headers to responses
type SecurityHeaders struct {
	// CSPEnabled enables Content-Security-Policy header
	CSPEnabled bool
	// CSPDirectives is the Content-Security-Policy value
	CSPDirectives string
	// HSTSEnabled enables Strict-Transport-Security header
	HSTSEnabled bool
	// HSTSMaxAge is the max-age value for HSTS (in seconds)
	HSTSMaxAge int
	// XFrameOptions is the X-Frame-Options value (DENY, SAMEORIGIN)
	XFrameOptions string
	// XContentTypeOptions is the X-Content-Type-Options value
	XContentTypeOptions string
}

// DefaultSecurityHeaders returns the default security headers configuration
func DefaultSecurityHeaders() *SecurityHeaders {
	return &SecurityHeaders{
		CSPEnabled:          true,
		CSPDirectives:       "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none';",
		HSTSEnabled:         true,
		HSTSMaxAge:          31536000, // 1 year
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
	}
}

// AuthEndpointSecurityHeaders returns security headers for auth endpoints
func AuthEndpointSecurityHeaders() *SecurityHeaders {
	return &SecurityHeaders{
		CSPEnabled:          true,
		CSPDirectives:       "default-src 'self'; script-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self';",
		HSTSEnabled:         true,
		HSTSMaxAge:          31536000,
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
	}
}

// Apply adds security headers to the response
func (sh *SecurityHeaders) Apply(w http.ResponseWriter) {
	if sh == nil {
		return
	}

	// Content-Security-Policy
	if sh.CSPEnabled && sh.CSPDirectives != "" {
		w.Header().Set("Content-Security-Policy", sh.CSPDirectives)
	}

	// Strict-Transport-Security
	if sh.HSTSEnabled && sh.HSTSMaxAge > 0 {
		w.Header().Set("Strict-Transport-Security", "max-age="+string(rune(sh.HSTSMaxAge))+"; includeSubDomains")
	}

	// X-Frame-Options
	if sh.XFrameOptions != "" {
		w.Header().Set("X-Frame-Options", sh.XFrameOptions)
	}

	// X-Content-Type-Options
	if sh.XContentTypeOptions != "" {
		w.Header().Set("X-Content-Type-Options", sh.XContentTypeOptions)
	}

	// Additional security headers
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}

// SetSecurityHeaders is a convenience function to set default security headers
func SetSecurityHeaders(w http.ResponseWriter) {
	DefaultSecurityHeaders().Apply(w)
}

// SetAuthSecurityHeaders is a convenience function to set auth endpoint security headers
func SetAuthSecurityHeaders(w http.ResponseWriter) {
	AuthEndpointSecurityHeaders().Apply(w)
}

// SecurityHeadersMiddleware is an http.Handler wrapper that adds security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	headers := DefaultSecurityHeaders()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers.Apply(w)
		next.ServeHTTP(w, r)
	})
}

// NoCacheHeaders sets headers to prevent caching (useful for auth responses)
func NoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// CORSHeaders sets CORS headers for API responses
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-API-Key", "X-Tunnel-API-Key"},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

// Apply sets CORS headers on the response
func (c *CORSConfig) Apply(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	// Check if origin is allowed
	allowed := false
	for _, o := range c.AllowOrigins {
		if o == "*" || o == origin {
			allowed = true
			break
		}
	}

	if !allowed {
		return
	}

	if c.AllowOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	if len(c.AllowMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", joinStrings(c.AllowMethods, ", "))
	}

	if len(c.AllowHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", joinStrings(c.AllowHeaders, ", "))
	}

	if c.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if c.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", string(rune(c.MaxAge)))
	}
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
