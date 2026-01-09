package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/policy"
	"github.com/niekvdm/digit-link/internal/protocol"
	"github.com/niekvdm/digit-link/internal/tunnel"
)

// isWebSocketUpgrade checks if the request is a WebSocket upgrade request
func isWebSocketUpgrade(r *http.Request) bool {
	connection := r.Header.Get("Connection")
	upgrade := r.Header.Get("Upgrade")
	return strings.Contains(strings.ToLower(connection), "upgrade") &&
		strings.EqualFold(upgrade, "websocket")
}

// pipe copies data bidirectionally between two connections
func pipe(conn1, conn2 net.Conn) (int64, int64) {
	var wg sync.WaitGroup
	var bytes1to2, bytes2to1 int64

	wg.Add(2)

	go func() {
		defer wg.Done()
		bytes1to2, _ = io.Copy(conn2, conn1)
		if c, ok := conn2.(interface{ CloseWrite() error }); ok {
			c.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		bytes2to1, _ = io.Copy(conn1, conn2)
		if c, ok := conn1.(interface{ CloseWrite() error }); ok {
			c.CloseWrite()
		}
	}()

	wg.Wait()
	return bytes1to2, bytes2to1
}

// Server manages tunnel connections and HTTP routing
type Server struct {
	domain   string
	scheme   string // URL scheme (http or https)
	secret   string // Legacy secret for backward compatibility
	db       *db.DB
	tunnels  map[string]*Tunnel
	mu       sync.RWMutex
	upgrader websocket.Upgrader

	// Auth middleware for tunnel-level authentication
	authMiddleware *AuthMiddleware

	// OIDC handler for OIDC authentication
	oidcHandler *auth.OIDCAuthHandler

	// Rate limiter for login endpoints
	loginRateLimiter *auth.RateLimiter

	// Usage tracking and quota enforcement
	usageCache   *UsageCache
	quotaChecker *QuotaChecker

	// TCP tunnel listener (yamux-based)
	tunnelListener *TunnelListener
}

// New creates a new tunnel server
func New(domain, scheme, secret string, database *db.DB) *Server {
	if scheme == "" {
		scheme = "https"
	}
	s := &Server{
		domain:  domain,
		scheme:  scheme,
		secret:  secret,
		db:      database,
		tunnels: make(map[string]*Tunnel),
	}

	// Initialize WebSocket upgrader with origin validation
	s.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Allow requests without Origin header (non-browser clients like CLI tools)
			if origin == "" {
				return true
			}

			// Check if origin matches the expected domain (with any subdomain)
			// Allow: https://link.digit.zone, https://subdomain.link.digit.zone
			allowedOrigins := []string{
				scheme + "://" + domain,
				scheme + "://localhost",
				"http://localhost", // Allow localhost for development
			}

			for _, allowed := range allowedOrigins {
				if strings.HasPrefix(origin, allowed) {
					return true
				}
			}

			// Also allow any subdomain of the main domain
			if strings.Contains(origin, "."+domain) {
				return true
			}

			log.Printf("WebSocket connection rejected: origin %s not allowed", origin)
			return false
		},
		ReadBufferSize:    1024 * 64,
		WriteBufferSize:   1024 * 64,
		EnableCompression: true, // Per-message compression for faster transmission
	}

	// Initialize auth handlers if database is available
	if database != nil {
		s.authMiddleware = NewAuthMiddleware(database, WithDefaultDeny(true))
		s.oidcHandler = auth.NewOIDCAuthHandler(database, domain)
		// Initialize rate limiter for login endpoints with stricter settings
		s.loginRateLimiter = auth.NewRateLimiter(database, auth.RateLimiterConfig{
			WindowDuration:  15 * time.Minute,
			MaxAttempts:     5, // Stricter than default for login
			BlockDuration:   30 * time.Minute,
			CleanupInterval: 5 * time.Minute,
		})

		// Initialize usage tracking and quota enforcement
		s.usageCache = NewUsageCache(database)
		s.usageCache.Start()
		s.quotaChecker = NewQuotaChecker(s.usageCache, database)
	}

	return s
}

// ServeHTTP handles all incoming HTTP requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// WebSocket upgrade for tunnel clients
	if r.URL.Path == "/_tunnel" {
		s.handleWebSocket(w, r)
		return
	}

	// Setup API endpoints
	if strings.HasPrefix(r.URL.Path, "/setup/") {
		s.handleSetup(w, r)
		return
	}

	// Authentication endpoints (admin dashboard auth)
	if strings.HasPrefix(r.URL.Path, "/auth/") {
		s.handleAuth(w, r)
		return
	}

	// For main domain requests, distinguish between API calls and SPA navigation
	// API calls have auth headers; browser navigation does not
	isMainDomain := s.extractSubdomain(r.Host) == ""
	if isMainDomain {
		// Public API endpoints (no auth required) - only on main domain
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.handlePublicAPI(w, r)
			return
		}

		// Check if this is an API request (has auth headers) or browser navigation
		hasAuthHeader := r.Header.Get("X-Admin-Token") != "" ||
			strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")

		// If no auth header, serve the SPA for client-side routing
		// This handles browser refresh on routes like /admin/accounts
		if !hasAuthHeader {
			s.serveDashboard(w, r)
			return
		}
	}

	// Admin API endpoints
	if strings.HasPrefix(r.URL.Path, "/admin/") {
		s.handleAdmin(w, r)
		return
	}

	// Org portal API endpoints
	if strings.HasPrefix(r.URL.Path, "/org/") {
		s.handleOrg(w, r)
		return
	}

	// Static files for dashboard (on main domain) - fallback for any remaining main domain requests
	if isMainDomain {
		s.serveDashboard(w, r)
		return
	}

	// Extract subdomain from Host header
	subdomain := s.extractSubdomain(r.Host)

	// Handle tunnel-level auth endpoints (mounted on subdomain)
	if strings.HasPrefix(r.URL.Path, "/__auth/") {
		s.handleTunnelAuth(w, r, subdomain)
		return
	}

	// Handle CORS preflight at tunnel level FIRST (before tunnel lookup)
	// This ensures CORS works even if tunnel lookup has issues
	if r.Method == http.MethodOptions {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Find tunnel for subdomain - check WebSocket tunnels first
	s.mu.RLock()
	wsTunnel, wsOk := s.tunnels[subdomain]
	s.mu.RUnlock()

	// Check TCP tunnels if no WebSocket tunnel found
	var tcpSession *tunnel.Session
	var tcpOk bool
	if !wsOk && s.tunnelListener != nil {
		tcpSession, tcpOk = s.tunnelListener.GetSession(subdomain)
	}

	if !wsOk && !tcpOk {
		http.Error(w, fmt.Sprintf("Tunnel '%s' not found", subdomain), http.StatusNotFound)
		return
	}

	// Apply tunnel-level authentication if middleware is configured
	if s.authMiddleware != nil {
		result, authCtx := s.authMiddleware.AuthenticateRequest(w, r, subdomain)

		// Get the effective policy from context for challenge handling
		effectivePolicy := GetEffectivePolicyFromContext(r)

		if !s.authMiddleware.HandleAuthResult(w, r, result, effectivePolicy, authCtx) {
			// Auth failed, response already sent
			return
		}
	}

	// Forward request through appropriate tunnel type
	if wsOk {
		s.forwardRequest(w, r, wsTunnel)
	} else {
		s.forwardRequestViaTCP(w, r, tcpSession, subdomain)
	}
}

// handlePublicAPI handles public API endpoints that don't require authentication
func (s *Server) handlePublicAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")

	switch {
	case path == "/plans" && r.Method == http.MethodGet:
		s.handlePublicListPlans(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handlePublicListPlans returns all plans for public display (pricing page)
func (s *Server) handlePublicListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.db.ListPlans()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plans": plans,
	})
}

// serveDashboard serves the Vue SPA - all routes return index.html except static assets
func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check if initial setup is needed - redirect to /setup route
	if s.NeedsSetup() {
		if path == "/" {
			http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
			return
		}
	}

	// Try to serve static assets (JS, CSS, images, etc.)
	if strings.HasPrefix(path, "/assets/") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".css") ||
		strings.HasSuffix(path, ".svg") ||
		strings.HasSuffix(path, ".png") ||
		strings.HasSuffix(path, ".ico") ||
		strings.HasSuffix(path, ".woff") ||
		strings.HasSuffix(path, ".woff2") {
		content, contentType, found := getStaticFile(path)
		if found {
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.Write(content)
			return
		}
	}

	// For all other routes, serve index.html (Vue Router will handle routing)
	content, contentType, found := getStaticFile("/index.html")
	if found {
		w.Header().Set("Content-Type", contentType)
		w.Write(content)
		return
	}

	// Fallback to basic status page if index.html not found
	s.mu.RLock()
	tunnelCount := len(s.tunnels)
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>digit-link</title></head>
<body>
<h1>digit-link tunnel server</h1>
<p>Connect with: <code>digit-link --server %s --subdomain &lt;name&gt; --port &lt;port&gt; --token &lt;token&gt;</code></p>
<p>Active tunnels: %d</p>
</body>
</html>`, s.domain, tunnelCount)
}

// GetActiveTunnels returns a list of active tunnels (for admin API)
func (s *Server) GetActiveTunnels() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tunnels := make([]map[string]interface{}, 0, len(s.tunnels))
	for subdomain, tunnel := range s.tunnels {
		tunnels = append(tunnels, map[string]interface{}{
			"subdomain": subdomain,
			"url":       fmt.Sprintf("%s://%s.%s", s.scheme, subdomain, s.domain),
			"createdAt": tunnel.CreatedAt,
		})
	}
	return tunnels
}

// DB returns the database instance
func (s *Server) DB() *db.DB {
	return s.db
}

// extractSubdomain extracts the subdomain from the host
func (s *Server) extractSubdomain(host string) string {
	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Also remove port from domain for comparison
	domain := s.domain
	if idx := strings.LastIndex(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Check if it's a subdomain of our domain
	if !strings.HasSuffix(host, domain) {
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+domain)
	if subdomain == host || subdomain == "" {
		return ""
	}

	return subdomain
}

// handleWebSocket handles WebSocket connections from tunnel clients
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get client IP for whitelist check
	clientIP := auth.GetClientIP(r)

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Enable TCP_NODELAY to disable Nagle's algorithm for lower latency
	if tcpConn := conn.UnderlyingConn(); tcpConn != nil {
		if tc, ok := tcpConn.(*net.TCPConn); ok {
			tc.SetNoDelay(true)
		}
	}

	// Wait for registration message
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Printf("Failed to read registration: %v", err)
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})

	var message protocol.Message
	if err := json.Unmarshal(msg, &message); err != nil {
		log.Printf("Invalid registration message: %v", err)
		conn.Close()
		return
	}

	if message.Type != protocol.TypeRegisterRequest {
		log.Printf("Expected register_request, got %s", message.Type)
		conn.Close()
		return
	}

	// Parse registration payload
	payloadBytes, _ := json.Marshal(message.Payload)
	var regReq protocol.RegisterRequest
	if err := json.Unmarshal(payloadBytes, &regReq); err != nil {
		log.Printf("Invalid registration payload: %v", err)
		conn.Close()
		return
	}

	// Authentication result tracking
	var account *db.Account
	var apiKey *db.APIKey
	var app *db.Application
	var orgID string

	if s.db != nil {
		// Try token-based authentication first
		if regReq.Token == "" {
			// Fallback to legacy secret if no token provided
			if s.secret != "" && regReq.Secret != s.secret {
				log.Printf("Authentication failed for subdomain %s from %s: no valid token or secret", regReq.Subdomain, clientIP)
				s.sendRegisterResponse(conn, false, "", "", "Authentication required: provide a valid token")
				conn.Close()
				return
			}
			// Legacy mode without token - skip account/IP checks if secret matches
			if s.secret == "" {
				log.Printf("Authentication failed for subdomain %s from %s: no token provided", regReq.Subdomain, clientIP)
				s.sendRegisterResponse(conn, false, "", "", "Authentication required: provide a valid token")
				conn.Close()
				return
			}
		} else {
			// Try to validate as API key first
			apiKeyHash := db.HashAPIKey(regReq.Token)
			apiKey, err = s.db.GetAPIKeyByHash(apiKeyHash)
			if err != nil {
				log.Printf("Database error during API key lookup: %v", err)
				s.sendRegisterResponse(conn, false, "", "", "Internal server error")
				conn.Close()
				return
			}

			if apiKey != nil {
				// Check if key is expired
				if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
					log.Printf("Authentication failed for subdomain %s from %s: API key expired", regReq.Subdomain, clientIP)
					s.sendRegisterResponse(conn, false, "", "", "API key has expired")
					conn.Close()
					return
				}

				// Handle app-specific API key
				if apiKey.KeyType == db.KeyTypeApp && apiKey.AppID != nil {
					// Get the app for this key
					app, err = s.db.GetApplicationByID(*apiKey.AppID)
					if err != nil || app == nil {
						log.Printf("Authentication failed for subdomain %s from %s: app not found for API key", regReq.Subdomain, clientIP)
						s.sendRegisterResponse(conn, false, "", "", "Application not found for API key")
						conn.Close()
						return
					}

					// For app API keys, enforce the subdomain must match the app's subdomain
					if regReq.Subdomain != "" && strings.ToLower(regReq.Subdomain) != app.Subdomain {
						log.Printf("Authentication failed for subdomain %s from %s: app API key can only connect to %s", regReq.Subdomain, clientIP, app.Subdomain)
						s.sendRegisterResponse(conn, false, "", "", fmt.Sprintf("This API key can only connect to subdomain '%s'", app.Subdomain))
						conn.Close()
						return
					}

					// Set subdomain to the app's subdomain
					regReq.Subdomain = app.Subdomain
					orgID = app.OrgID

					// Check app-level whitelist
					whitelisted, err := s.db.IsIPWhitelistedForApp(clientIP, app.ID)
					if err != nil {
						log.Printf("Whitelist check error: %v", err)
						s.sendRegisterResponse(conn, false, "", "", "Internal server error")
						conn.Close()
						return
					}
					if !whitelisted {
						log.Printf("Connection rejected for app %s (%s): IP %s not whitelisted", app.Name, regReq.Subdomain, clientIP)
						s.sendRegisterResponse(conn, false, "", "", "IP address not whitelisted")
						conn.Close()
						return
					}
				} else if apiKey.OrgID != nil {
					// Account-level API key (for random subdomains)
					orgID = *apiKey.OrgID

					// Check org-level whitelist
					whitelisted, err := s.db.IsIPWhitelistedForOrg(clientIP, orgID)
					if err != nil {
						log.Printf("Whitelist check error: %v", err)
						s.sendRegisterResponse(conn, false, "", "", "Internal server error")
						conn.Close()
						return
					}
					if !whitelisted {
						log.Printf("Connection rejected for org %s (%s): IP %s not whitelisted", orgID, regReq.Subdomain, clientIP)
						s.sendRegisterResponse(conn, false, "", "", "IP address not whitelisted")
						conn.Close()
						return
					}
				}

				// Update API key last used timestamp
				s.db.UpdateAPIKeyLastUsed(apiKey.ID)
			} else {
				// Not an API key, try account token
				tokenHash := auth.HashToken(regReq.Token)
				account, err = s.db.GetAccountByTokenHash(tokenHash)
				if err != nil {
					log.Printf("Database error during auth: %v", err)
					s.sendRegisterResponse(conn, false, "", "", "Internal server error")
					conn.Close()
					return
				}
				if account == nil {
					log.Printf("Authentication failed for subdomain %s from %s: invalid token", regReq.Subdomain, clientIP)
					s.sendRegisterResponse(conn, false, "", "", "Invalid token")
					conn.Close()
					return
				}

				orgID = account.OrgID

				// Check IP whitelist
				whitelisted, err := s.db.IsIPWhitelistedForAccount(clientIP, account.ID)
				if err != nil {
					log.Printf("Whitelist check error: %v", err)
					s.sendRegisterResponse(conn, false, "", "", "Internal server error")
					conn.Close()
					return
				}
				if !whitelisted {
					log.Printf("Connection rejected for %s (%s): IP %s not whitelisted", account.Username, regReq.Subdomain, clientIP)
					s.sendRegisterResponse(conn, false, "", "", "IP address not whitelisted")
					conn.Close()
					return
				}

				// Update last used timestamp
				s.db.UpdateAccountLastUsed(account.ID)
			}
		}
	} else {
		// No database - legacy mode with secret only
		if s.secret != "" && regReq.Secret != s.secret {
			s.sendRegisterResponse(conn, false, "", "", "Invalid secret")
			conn.Close()
			return
		}
	}

	// Validate or generate subdomain
	subdomain := strings.ToLower(regReq.Subdomain)
	if subdomain == "" {
		// Generate a random subdomain
		subdomain = generateRandomSubdomain()
		log.Printf("Generated random subdomain: %s", subdomain)
	} else if !isValidSubdomain(subdomain) {
		s.sendRegisterResponse(conn, false, "", "", "Invalid subdomain")
		conn.Close()
		return
	}

	// Check if subdomain is already in use
	s.mu.Lock()
	if _, exists := s.tunnels[subdomain]; exists {
		s.mu.Unlock()
		s.sendRegisterResponse(conn, false, "", "", "Subdomain already in use")
		conn.Close()
		return
	}

	// Check quota before registering tunnel
	if s.quotaChecker != nil && orgID != "" {
		allowed, reason := s.quotaChecker.CanConnectTunnel(orgID)
		if !allowed {
			s.mu.Unlock()
			s.sendRegisterResponse(conn, false, "", "", fmt.Sprintf("Quota exceeded: %s", reason))
			conn.Close()
			return
		}
		// Track concurrent tunnel increase
		s.usageCache.IncrementConcurrentTunnels(orgID)
	}

	// Register tunnel with context
	var appID string
	if app != nil {
		appID = app.ID
	}
	tunnel := NewTunnelWithContext(subdomain, conn, "", orgID, appID, app)
	if account != nil {
		tunnel.AccountID = account.ID
	}
	s.tunnels[subdomain] = tunnel
	s.mu.Unlock()

	// Record tunnel in database
	var tunnelRecordID string
	if s.db != nil {
		var accountIDForRecord string
		if account != nil {
			accountIDForRecord = account.ID
		}
		// Always create tunnel record (accountID can be empty for API key auth)
		tunnelRecord, err := s.db.CreateTunnel(accountIDForRecord, subdomain, clientIP)
		if err != nil {
			log.Printf("Failed to record tunnel: %v", err)
		} else {
			tunnelRecordID = tunnelRecord.ID
			tunnel.RecordID = tunnelRecordID // Store in tunnel for stats tracking
			// Update with app_id if applicable
			if appID != "" {
				s.db.UpdateTunnelAppID(tunnelRecordID, appID)
			}
		}
	}

	url := fmt.Sprintf("%s://%s.%s", s.scheme, subdomain, s.domain)
	if account != nil {
		log.Printf("Tunnel registered: %s -> %s (user: %s, ip: %s)", subdomain, url, account.Username, clientIP)
	} else if apiKey != nil {
		keyType := "account"
		if apiKey.KeyType == db.KeyTypeApp {
			keyType = "app"
		}
		log.Printf("Tunnel registered: %s -> %s (api_key: %s..., type: %s, ip: %s)", subdomain, url, apiKey.KeyPrefix, keyType, clientIP)
	} else {
		log.Printf("Tunnel registered: %s -> %s (legacy auth, ip: %s)", subdomain, url, clientIP)
	}

	// Send success response
	s.sendRegisterResponse(conn, true, subdomain, url, "")

	// Handle incoming messages (responses from client)
	tunnelStartTime := time.Now()
	s.handleTunnelMessages(tunnel)

	// Cleanup on disconnect
	s.mu.Lock()
	delete(s.tunnels, subdomain)
	s.mu.Unlock()
	tunnel.Close()

	// Track usage on disconnect
	if s.usageCache != nil && orgID != "" {
		// Decrement concurrent tunnels
		s.usageCache.DecrementConcurrentTunnels(orgID)
		// Record tunnel duration
		tunnelDuration := time.Since(tunnelStartTime)
		s.usageCache.RecordTunnelTime(orgID, int64(tunnelDuration.Seconds()))
	}

	// Close tunnel record in database
	if s.db != nil && tunnelRecordID != "" {
		s.db.CloseTunnel(tunnelRecordID)
	}

	log.Printf("Tunnel disconnected: %s", subdomain)
}

// sendRegisterResponse sends a registration response to the client
func (s *Server) sendRegisterResponse(conn *websocket.Conn, success bool, subdomain, url, errMsg string) {
	resp := protocol.Message{
		Type: protocol.TypeRegisterResponse,
		Payload: protocol.RegisterResponse{
			Success:   success,
			Subdomain: subdomain,
			URL:       url,
			Error:     errMsg,
		},
	}
	data, _ := json.Marshal(resp)
	conn.WriteMessage(websocket.TextMessage, data)
}

// pongWait is the time allowed to read the next pong message from the peer
const pongWait = 60 * time.Second

// handleTunnelMessages handles messages from a connected tunnel client
func (s *Server) handleTunnelMessages(tunnel *Tunnel) {
	// Set initial read deadline
	tunnel.Conn.SetReadDeadline(time.Now().Add(pongWait))

	// Set pong handler to reset the read deadline on each pong
	tunnel.Conn.SetPongHandler(func(string) error {
		tunnel.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := tunnel.Conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("Tunnel read error (%s): %v", tunnel.Subdomain, err)
			}
			return
		}

		// Reset read deadline on any message received
		tunnel.Conn.SetReadDeadline(time.Now().Add(pongWait))

		// Use TypedMessage to extract type without fully parsing payload
		var message protocol.TypedMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Invalid message from tunnel: %v", err)
			continue
		}

		switch message.Type {
		case protocol.TypeHTTPResponse:
			// Forward raw message to waiting request handler - avoids re-parsing
			if ch, ok := tunnel.GetResponseChannel(s.extractRequestIDFromRaw(message.Payload)); ok {
				ch <- msg
			}
		case protocol.TypePong:
			// Heartbeat response - deadline already reset above
		}
	}
}

// extractRequestIDFromRaw extracts the request ID from raw JSON payload
func (s *Server) extractRequestIDFromRaw(payload json.RawMessage) string {
	// Quick extraction of just the ID field without full unmarshal
	var idExtract struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &idExtract); err != nil {
		return ""
	}
	return idExtract.ID
}

// extractRequestID extracts the request ID from a response payload (legacy)
func (s *Server) extractRequestID(payload interface{}) string {
	if m, ok := payload.(map[string]interface{}); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return ""
}

// forwardRequest forwards an HTTP request through the tunnel
func (s *Server) forwardRequest(w http.ResponseWriter, r *http.Request, tunnel *Tunnel) {
	// Check quota before processing request
	if s.quotaChecker != nil && tunnel.OrgID != "" {
		allowed, reason := s.quotaChecker.CanProcessRequest(tunnel.OrgID)
		if !allowed {
			// Add quota headers to response
			headers := s.quotaChecker.GetQuotaHeaders(tunnel.OrgID, QuotaRequests)
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			w.Header().Set("Retry-After", "86400") // Retry after 1 day (end of billing period)
			http.Error(w, fmt.Sprintf("Quota exceeded: %s", reason), http.StatusTooManyRequests)
			return
		}
	}

	requestID := uuid.New().String()

	// Build HTTP request message
	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = values[0]
	}

	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}

	httpReq := protocol.HTTPRequest{
		ID:      requestID,
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: headers,
		Body:    body,
	}

	msg := protocol.Message{
		Type:    protocol.TypeHTTPRequest,
		Payload: httpReq,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Track bytes sent (request size)
	bytesSent := int64(len(data))

	// Create response channel
	responseCh := tunnel.AddResponseChannel(requestID)
	defer tunnel.RemoveResponseChannel(requestID)

	// Send request to tunnel client
	if err := tunnel.WriteMessage(websocket.TextMessage, data); err != nil {
		http.Error(w, "Tunnel error", http.StatusBadGateway)
		return
	}

	// Wait for response with timeout
	select {
	case responseData := <-responseCh:
		// Track bytes received (response size)
		bytesReceived := int64(len(responseData))

		// Update tunnel stats in database
		if s.db != nil && tunnel.RecordID != "" {
			go s.db.UpdateTunnelStatsWithRequests(tunnel.RecordID, bytesSent, bytesReceived, 1)
		}

		// Update usage cache for quota tracking
		if s.usageCache != nil && tunnel.OrgID != "" {
			s.usageCache.RecordBandwidth(tunnel.OrgID, bytesSent+bytesReceived)
			s.usageCache.RecordRequest(tunnel.OrgID)
		}

		// Use TypedMessage to parse directly without double serialization
		var respMsg protocol.TypedMessage
		if err := json.Unmarshal(responseData, &respMsg); err != nil {
			http.Error(w, "Invalid response", http.StatusBadGateway)
			return
		}

		// Parse response payload directly from raw JSON
		var httpResp protocol.HTTPResponse
		if err := json.Unmarshal(respMsg.Payload, &httpResp); err != nil {
			http.Error(w, "Invalid response payload", http.StatusBadGateway)
			return
		}

		// Write response headers
		for key, value := range httpResp.Headers {
			w.Header().Set(key, value)
		}

		// Add CORS headers if Origin was present in request
		addCORSHeaders(w, r)

		w.WriteHeader(httpResp.StatusCode)
		if len(httpResp.Body) > 0 {
			w.Write(httpResp.Body)
		}

	case <-time.After(5 * time.Minute):
		http.Error(w, "Tunnel timeout", http.StatusGatewayTimeout)
	}
}

// forwardRequestViaTCP forwards an HTTP request through a TCP/yamux tunnel
func (s *Server) forwardRequestViaTCP(w http.ResponseWriter, r *http.Request, session *tunnel.Session, subdomain string) {
	// Get org ID for quota checking
	accountID, orgID, _ := session.GetAccountInfo()

	// Check quota before processing request
	if s.quotaChecker != nil && orgID != "" {
		allowed, reason := s.quotaChecker.CanProcessRequest(orgID)
		if !allowed {
			headers := s.quotaChecker.GetQuotaHeaders(orgID, QuotaRequests)
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			w.Header().Set("Retry-After", "86400")
			http.Error(w, fmt.Sprintf("Quota exceeded: %s", reason), http.StatusTooManyRequests)
			return
		}
	}

	// Check if this is a WebSocket upgrade request
	isWS := isWebSocketUpgrade(r)

	// Open a new yamux stream for this request
	stream, err := session.Open()
	if err != nil {
		log.Printf("Failed to open yamux stream for %s: %v", subdomain, err)
		http.Error(w, "Tunnel unavailable", http.StatusBadGateway)
		return
	}

	// For regular HTTP, defer close. For WebSocket, we'll handle it after piping
	if !isWS {
		defer stream.Close()
	}

	requestID := uuid.New().String()

	// Build request headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = values[0]
	}

	// Read request body
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}

	// Create request frame
	reqFrame := tunnel.RequestFrame{
		ID:        requestID,
		Subdomain: subdomain,
		Method:    r.Method,
		Path:      r.URL.RequestURI(),
		Headers:   headers,
		Body:      body,
	}

	// Send request frame
	if err := tunnel.WriteFrame(stream, &reqFrame); err != nil {
		log.Printf("Failed to write request frame for %s: %v", subdomain, err)
		http.Error(w, "Tunnel error", http.StatusBadGateway)
		if isWS {
			stream.Close()
		}
		return
	}

	// Track bytes sent
	bytesSent := int64(len(body) + 500) // Approximate frame overhead

	// Read response frame with timeout
	stream.SetReadDeadline(time.Now().Add(5 * time.Minute))
	respFrame, err := tunnel.ReadFrame[tunnel.ResponseFrame](stream)
	if err != nil {
		log.Printf("Failed to read response frame for %s: %v", subdomain, err)
		http.Error(w, "Tunnel timeout or error", http.StatusGatewayTimeout)
		if isWS {
			stream.Close()
		}
		return
	}

	// Clear read deadline for WebSocket piping
	if isWS {
		stream.SetReadDeadline(time.Time{})
	}

	// Track bytes received
	bytesReceived := int64(len(respFrame.Body) + 500) // Approximate frame overhead

	// Update usage tracking
	if s.usageCache != nil && orgID != "" {
		s.usageCache.RecordBandwidth(orgID, bytesSent+bytesReceived)
		s.usageCache.RecordRequest(orgID)
	}

	// Log request (optional - for debugging)
	_ = accountID // Silence unused variable if not logging

	// Handle WebSocket upgrade (101 Switching Protocols)
	if isWS && respFrame.Status == http.StatusSwitchingProtocols {
		s.handleWebSocketUpgrade(w, r, stream, respFrame, orgID)
		return
	}

	// Regular HTTP response
	// Write response headers
	for key, value := range respFrame.Headers {
		w.Header().Set(key, value)
	}

	// Add CORS headers if Origin was present in request
	addCORSHeaders(w, r)

	w.WriteHeader(respFrame.Status)
	if len(respFrame.Body) > 0 {
		w.Write(respFrame.Body)
	}

	// Close stream for WebSocket requests that didn't get 101
	if isWS {
		stream.Close()
	}
}

// handleWebSocketUpgrade handles the WebSocket upgrade response and pipes data bidirectionally
func (s *Server) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request, stream net.Conn, respFrame *tunnel.ResponseFrame, orgID string) {
	// Hijack the HTTP connection to get raw TCP access
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("WebSocket upgrade failed: ResponseWriter does not support hijacking")
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		stream.Close()
		return
	}

	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		log.Printf("WebSocket upgrade failed: hijack error: %v", err)
		stream.Close()
		return
	}

	// Write the 101 response to the client
	var responseBuf strings.Builder
	responseBuf.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	for key, value := range respFrame.Headers {
		responseBuf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	responseBuf.WriteString("\r\n")

	if _, err := clientConn.Write([]byte(responseBuf.String())); err != nil {
		log.Printf("WebSocket upgrade failed: error writing 101 response: %v", err)
		clientConn.Close()
		stream.Close()
		return
	}

	// Check if there's any buffered data that needs to be written to the stream first
	if clientBuf.Reader.Buffered() > 0 {
		buffered := make([]byte, clientBuf.Reader.Buffered())
		clientBuf.Read(buffered)
		stream.Write(buffered)
	}

	// Pipe data bidirectionally between client and tunnel stream
	// This blocks until one side closes
	bytesSent, bytesRecv := pipe(clientConn, stream)

	// Update usage tracking for WebSocket traffic
	if s.usageCache != nil && orgID != "" {
		s.usageCache.RecordBandwidth(orgID, bytesSent+bytesRecv)
	}

	// Cleanup
	clientConn.Close()
	stream.Close()
}

// addCORSHeaders adds CORS headers to response if Origin header was present in request
func addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	// Only add if not already set by backend
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

// reservedSubdomains contains subdomains that cannot be registered by users
// to prevent confusion and potential security issues
var reservedSubdomains = map[string]bool{
	"admin":     true,
	"api":       true,
	"auth":      true,
	"www":       true,
	"mail":      true,
	"ftp":       true,
	"static":    true,
	"assets":    true,
	"cdn":       true,
	"health":    true,
	"status":    true,
	"setup":     true,
	"login":     true,
	"logout":    true,
	"register":  true,
	"app":       true,
	"apps":      true,
	"dashboard": true,
	"console":   true,
	"portal":    true,
	"org":       true,
	"internal":  true,
	"system":    true,
	"root":      true,
	"test":      true,
	"dev":       true,
	"staging":   true,
	"prod":      true,
}

// isValidSubdomain checks if a subdomain name is valid
func isValidSubdomain(s string) bool {
	if len(s) < 1 || len(s) > 63 {
		return false
	}

	// Check against reserved subdomains
	if reservedSubdomains[s] {
		return false
	}

	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return s[0] != '-' && s[len(s)-1] != '-'
}

// generateRandomSubdomain creates a random subdomain using UUID
func generateRandomSubdomain() string {
	id := uuid.New().String()
	// Use first 8 characters of UUID for a short, unique subdomain
	return id[:8]
}

// Run starts the server on the specified port
func (s *Server) Run(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting digit-link server on %s (domain: %s)", addr, s.domain)

	// Start ping routine
	go s.pingRoutine()

	return http.ListenAndServe(addr, s)
}

// StartTunnelListener starts the TCP+TLS tunnel listener if configured
func (s *Server) StartTunnelListener() error {
	if !IsTunnelEnabled() {
		log.Printf("TCP tunnel listener disabled (set TUNNEL_ENABLED=true or provide TUNNEL_TLS_CERT/TUNNEL_TLS_KEY)")
		return nil
	}

	certFile := GetTunnelTLSCertFile()
	keyFile := GetTunnelTLSKeyFile()

	var tlsConfig *tls.Config
	var err error

	if certFile != "" && keyFile != "" {
		tlsConfig, err = tunnel.TLSServerConfig(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS config: %w", err)
		}
	}

	s.tunnelListener = NewTunnelListener(s, tlsConfig)
	port := GetTunnelPort()

	if err := s.tunnelListener.Start(port); err != nil {
		return fmt.Errorf("failed to start tunnel listener: %w", err)
	}

	return nil
}

// StopTunnelListener stops the TCP tunnel listener
func (s *Server) StopTunnelListener() error {
	if s.tunnelListener != nil {
		return s.tunnelListener.Stop()
	}
	return nil
}

// GetTunnelListener returns the tunnel listener (for request forwarding)
func (s *Server) GetTunnelListener() *TunnelListener {
	return s.tunnelListener
}

// pingPeriod is the period between pings (must be less than pongWait)
const pingPeriod = 30 * time.Second

// pingRoutine sends periodic pings to keep connections alive
func (s *Server) pingRoutine() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		tunnels := make([]*Tunnel, 0, len(s.tunnels))
		for _, t := range s.tunnels {
			tunnels = append(tunnels, t)
		}
		s.mu.RUnlock()

		for _, tunnel := range tunnels {
			// Send WebSocket ping frame (triggers pong response)
			if err := tunnel.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping to tunnel %s: %v", tunnel.Subdomain, err)
			}
		}
	}
}

// GetDomain returns the server domain from environment or default
func GetDomain() string {
	if domain := os.Getenv("DOMAIN"); domain != "" {
		return domain
	}
	return "link.digit.zone"
}

// GetScheme returns the URL scheme from environment or default (https)
func GetScheme() string {
	if scheme := os.Getenv("SCHEME"); scheme != "" {
		return scheme
	}
	return "https"
}

// GetSecret returns the server secret from environment
func GetSecret() string {
	return os.Getenv("SECRET")
}

// GetPort returns the server port from environment or default
func GetPort() int {
	if port := os.Getenv("PORT"); port != "" {
		var p int
		fmt.Sscanf(port, "%d", &p)
		if p > 0 {
			return p
		}
	}
	return 8080
}

// handleTunnelAuth handles tunnel-level authentication endpoints
// These are mounted on subdomain paths like /__auth/login, /__auth/callback, etc.
func (s *Server) handleTunnelAuth(w http.ResponseWriter, r *http.Request, subdomain string) {
	// Set security headers for auth endpoints
	auth.SetAuthSecurityHeaders(w)
	auth.NoCacheHeaders(w)

	path := strings.TrimPrefix(r.URL.Path, "/__auth")

	switch path {
	case "/login":
		s.handleTunnelAuthLogin(w, r, subdomain)
	case "/callback":
		s.handleTunnelAuthCallback(w, r, subdomain)
	case "/logout":
		s.handleTunnelAuthLogout(w, r, subdomain)
	case "/health":
		s.handleTunnelAuthHealth(w, r, subdomain)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleTunnelAuthLogin handles the OIDC login flow
func (s *Server) handleTunnelAuthLogin(w http.ResponseWriter, r *http.Request, subdomain string) {
	if s.db == nil {
		http.Error(w, "Authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Look up the application and its auth policy
	app, err := s.db.GetApplicationBySubdomain(subdomain)
	if err != nil {
		log.Printf("Error looking up application for auth: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var orgID string
	if app != nil {
		orgID = app.OrgID
	}

	// Build auth context
	authCtx := &policy.AuthContext{
		Subdomain: subdomain,
	}
	if app != nil {
		authCtx.AppID = app.ID
		authCtx.OrgID = app.OrgID
		authCtx.App = app
		authCtx.IsPersistentApp = true
	}

	// Get effective policy
	var effectivePolicy *policy.EffectivePolicy
	if s.authMiddleware != nil && s.authMiddleware.policyLoader != nil {
		effectivePolicy, _, err = s.authMiddleware.policyLoader.LoadForSubdomain(subdomain)
		if err != nil {
			log.Printf("Error loading policy for subdomain %s: %v", subdomain, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	if effectivePolicy == nil {
		// Try to get org policy directly
		var oidcPolicy *db.OrgAuthPolicy
		if orgID != "" {
			oidcPolicy, err = s.db.GetOrgAuthPolicy(orgID)
			if err != nil {
				log.Printf("Error getting org auth policy: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		if oidcPolicy == nil || oidcPolicy.AuthType != db.AuthTypeOIDC {
			http.Error(w, "OIDC authentication not configured for this application", http.StatusNotImplemented)
			return
		}

		// Build effective policy from org policy
		effectivePolicy = &policy.EffectivePolicy{
			Type:  policy.AuthTypeOIDC,
			OrgID: orgID,
			OIDC: &policy.OIDCConfig{
				IssuerURL:      oidcPolicy.OIDCIssuerURL,
				ClientID:       oidcPolicy.OIDCClientID,
				ClientSecret:   oidcPolicy.OIDCClientSecretEnc, // Note: may need decryption
				Scopes:         oidcPolicy.OIDCScopes,
				AllowedDomains: oidcPolicy.OIDCAllowedDomains,
				RequiredClaims: oidcPolicy.OIDCRequiredClaims,
			},
		}
	}

	if effectivePolicy.Type != policy.AuthTypeOIDC || effectivePolicy.OIDC == nil {
		http.Error(w, "OIDC authentication not configured for this application", http.StatusNotImplemented)
		return
	}

	// Handle OIDC login (redirect URL is set per-request in HandleLogin)
	if s.oidcHandler != nil {
		s.oidcHandler.HandleLogin(w, r, effectivePolicy, authCtx)
	} else {
		http.Error(w, "OIDC handler not initialized", http.StatusInternalServerError)
	}
}

// handleTunnelAuthCallback handles the OIDC callback
func (s *Server) handleTunnelAuthCallback(w http.ResponseWriter, r *http.Request, subdomain string) {
	if s.db == nil || s.oidcHandler == nil {
		http.Error(w, "Authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Look up the application
	app, err := s.db.GetApplicationBySubdomain(subdomain)
	if err != nil {
		log.Printf("Error looking up application for auth callback: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var orgID string
	if app != nil {
		orgID = app.OrgID
	}

	// Build auth context
	authCtx := &policy.AuthContext{
		Subdomain: subdomain,
	}
	if app != nil {
		authCtx.AppID = app.ID
		authCtx.OrgID = app.OrgID
		authCtx.App = app
		authCtx.IsPersistentApp = true
	}

	// Get effective policy
	var effectivePolicy *policy.EffectivePolicy
	if s.authMiddleware != nil && s.authMiddleware.policyLoader != nil {
		effectivePolicy, _, err = s.authMiddleware.policyLoader.LoadForSubdomain(subdomain)
		if err != nil {
			log.Printf("Error loading policy for subdomain %s: %v", subdomain, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	if effectivePolicy == nil {
		// Try to get org policy directly
		var oidcPolicy *db.OrgAuthPolicy
		if orgID != "" {
			oidcPolicy, err = s.db.GetOrgAuthPolicy(orgID)
			if err != nil {
				log.Printf("Error getting org auth policy: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		if oidcPolicy == nil || oidcPolicy.AuthType != db.AuthTypeOIDC {
			http.Error(w, "OIDC authentication not configured", http.StatusNotImplemented)
			return
		}

		effectivePolicy = &policy.EffectivePolicy{
			Type:  policy.AuthTypeOIDC,
			OrgID: orgID,
			OIDC: &policy.OIDCConfig{
				IssuerURL:      oidcPolicy.OIDCIssuerURL,
				ClientID:       oidcPolicy.OIDCClientID,
				ClientSecret:   oidcPolicy.OIDCClientSecretEnc,
				Scopes:         oidcPolicy.OIDCScopes,
				AllowedDomains: oidcPolicy.OIDCAllowedDomains,
				RequiredClaims: oidcPolicy.OIDCRequiredClaims,
			},
		}
	}

	if effectivePolicy.Type != policy.AuthTypeOIDC || effectivePolicy.OIDC == nil {
		http.Error(w, "OIDC authentication not configured", http.StatusNotImplemented)
		return
	}

	s.oidcHandler.HandleCallback(w, r, effectivePolicy, authCtx)
}

// handleTunnelAuthLogout handles logout (clears session)
func (s *Server) handleTunnelAuthLogout(w http.ResponseWriter, r *http.Request, subdomain string) {
	if s.oidcHandler != nil {
		s.oidcHandler.HandleLogout(w, r)
	} else {
		// Fallback: Clear the session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "digit_link_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		// Get redirect URL
		redirectURL := r.URL.Query().Get("redirect")
		if redirectURL == "" {
			redirectURL = "/"
		}

		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// handleTunnelAuthHealth returns the health status of the auth system
func (s *Server) handleTunnelAuthHealth(w http.ResponseWriter, r *http.Request, subdomain string) {
	response := map[string]interface{}{
		"status":    "ok",
		"subdomain": subdomain,
	}

	if s.db != nil {
		// Check if auth is configured for this subdomain
		app, err := s.db.GetApplicationBySubdomain(subdomain)
		if err == nil && app != nil {
			response["appId"] = app.ID
			response["authMode"] = app.AuthMode

			// Check if policy exists
			if app.AuthMode == db.AuthModeCustom {
				hasPolicy, _ := s.db.HasAppAuthPolicy(app.ID)
				response["hasCustomPolicy"] = hasPolicy
			} else if app.AuthMode == db.AuthModeInherit {
				hasPolicy, _ := s.db.HasOrgAuthPolicy(app.OrgID)
				response["hasOrgPolicy"] = hasPolicy
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTunnelAuthContext returns the auth context for a tunnel
// This is used by the tunnel to know what org it belongs to
func (s *Server) GetTunnelAuthContext(subdomain string) *policy.AuthContext {
	if s.db == nil {
		return &policy.AuthContext{Subdomain: subdomain}
	}

	app, err := s.db.GetApplicationBySubdomain(subdomain)
	if err != nil || app == nil {
		return &policy.AuthContext{Subdomain: subdomain}
	}

	return &policy.AuthContext{
		Subdomain:       subdomain,
		AppID:           app.ID,
		OrgID:           app.OrgID,
		App:             app,
		IsPersistentApp: true,
	}
}
