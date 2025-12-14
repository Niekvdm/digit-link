package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/protocol"
)

// Server manages tunnel connections and HTTP routing
type Server struct {
	domain   string
	secret   string // Legacy secret for backward compatibility
	db       *db.DB
	tunnels  map[string]*Tunnel
	mu       sync.RWMutex
	upgrader websocket.Upgrader
}

// New creates a new tunnel server
func New(domain, secret string, database *db.DB) *Server {
	return &Server{
		domain:  domain,
		secret:  secret,
		db:      database,
		tunnels: make(map[string]*Tunnel),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024 * 64,
			WriteBufferSize: 1024 * 64,
		},
	}
}

// ServeHTTP handles all incoming HTTP requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check endpoint
	if r.URL.Path == "/health" {
		s.handleHealth(w, r)
		return
	}

	// WebSocket upgrade for tunnel clients
	if r.URL.Path == "/_tunnel" {
		s.handleWebSocket(w, r)
		return
	}

	// Setup endpoints (always available for status check)
	if strings.HasPrefix(r.URL.Path, "/setup") {
		s.handleSetup(w, r)
		return
	}

	// Admin API endpoints
	if strings.HasPrefix(r.URL.Path, "/admin/") {
		s.handleAdmin(w, r)
		return
	}

	// Static files for dashboard (on main domain)
	if s.extractSubdomain(r.Host) == "" {
		// Serve static dashboard files
		s.serveDashboard(w, r)
		return
	}

	// Extract subdomain from Host header
	subdomain := s.extractSubdomain(r.Host)

	// Find tunnel for subdomain
	s.mu.RLock()
	tunnel, ok := s.tunnels[subdomain]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("Tunnel '%s' not found", subdomain), http.StatusNotFound)
		return
	}

	// Forward request through tunnel
	s.forwardRequest(w, r, tunnel)
}

// handleHealth returns health and stats info
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tunnelCount := len(s.tunnels)
	s.mu.RUnlock()

	response := map[string]interface{}{
		"status":        "ok",
		"activeTunnels": tunnelCount,
	}

	if s.db != nil {
		if count, err := s.db.CountActiveAccounts(); err == nil {
			response["activeAccounts"] = count
		}
		if count, err := s.db.CountGlobalWhitelist(); err == nil {
			response["whitelistEntries"] = count
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// serveDashboard serves the static dashboard files or the default page
func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request) {
	// Check if initial setup is needed
	if s.NeedsSetup() {
		path := r.URL.Path
		// If requesting root or index, redirect to setup
		if path == "/" || path == "/index.html" {
			content, contentType, found := getStaticFile("/setup.html")
			if found {
				w.Header().Set("Content-Type", contentType)
				w.Write(content)
				return
			}
			// Fallback redirect
			http.Redirect(w, r, "/setup.html", http.StatusTemporaryRedirect)
			return
		}
	}

	// Try to serve static files from embedded public directory
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Check if file exists in public directory
	content, contentType, found := getStaticFile(path)
	if found {
		w.Header().Set("Content-Type", contentType)
		w.Write(content)
		return
	}

	// Fallback to basic status page
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
			"url":       fmt.Sprintf("https://%s.%s", subdomain, s.domain),
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

	// Check if it's a subdomain of our domain
	if !strings.HasSuffix(host, s.domain) {
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+s.domain)
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

	// Authenticate using token-based auth (preferred) or legacy secret
	var account *db.Account
	if s.db != nil {
		// Token-based authentication
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
			// Validate token
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
	} else {
		// No database - legacy mode with secret only
		if s.secret != "" && regReq.Secret != s.secret {
			s.sendRegisterResponse(conn, false, "", "", "Invalid secret")
			conn.Close()
			return
		}
	}

	// Validate subdomain
	subdomain := strings.ToLower(regReq.Subdomain)
	if subdomain == "" || !isValidSubdomain(subdomain) {
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

	// Register tunnel
	tunnel := NewTunnel(subdomain, conn)
	s.tunnels[subdomain] = tunnel
	s.mu.Unlock()

	// Record tunnel in database if we have an authenticated account
	var tunnelRecordID string
	if s.db != nil && account != nil {
		tunnelRecord, err := s.db.CreateTunnel(account.ID, subdomain, clientIP)
		if err != nil {
			log.Printf("Failed to record tunnel: %v", err)
		} else {
			tunnelRecordID = tunnelRecord.ID
		}
	}

	url := fmt.Sprintf("https://%s.%s", subdomain, s.domain)
	if account != nil {
		log.Printf("Tunnel registered: %s -> %s (user: %s, ip: %s)", subdomain, url, account.Username, clientIP)
	} else {
		log.Printf("Tunnel registered: %s -> %s (legacy auth, ip: %s)", subdomain, url, clientIP)
	}

	// Send success response
	s.sendRegisterResponse(conn, true, subdomain, url, "")

	// Handle incoming messages (responses from client)
	s.handleTunnelMessages(tunnel)

	// Cleanup on disconnect
	s.mu.Lock()
	delete(s.tunnels, subdomain)
	s.mu.Unlock()
	tunnel.Close()

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

// handleTunnelMessages handles messages from a connected tunnel client
func (s *Server) handleTunnelMessages(tunnel *Tunnel) {
	for {
		_, msg, err := tunnel.Conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("Tunnel read error (%s): %v", tunnel.Subdomain, err)
			}
			return
		}

		var message protocol.Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Invalid message from tunnel: %v", err)
			continue
		}

		switch message.Type {
		case protocol.TypeHTTPResponse:
			// Forward response to waiting request handler
			if ch, ok := tunnel.GetResponseChannel(s.extractRequestID(message.Payload)); ok {
				ch <- msg
			}
		case protocol.TypePong:
			// Heartbeat response, ignore
		}
	}
}

// extractRequestID extracts the request ID from a response payload
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

	// Create response channel
	responseCh := tunnel.AddResponseChannel(requestID)
	defer tunnel.RemoveResponseChannel(requestID)

	// Send request to tunnel client
	if err := tunnel.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		http.Error(w, "Tunnel error", http.StatusBadGateway)
		return
	}

	// Wait for response with timeout
	select {
	case responseData := <-responseCh:
		var respMsg protocol.Message
		if err := json.Unmarshal(responseData, &respMsg); err != nil {
			http.Error(w, "Invalid response", http.StatusBadGateway)
			return
		}

		// Parse response payload
		payloadBytes, _ := json.Marshal(respMsg.Payload)
		var httpResp protocol.HTTPResponse
		if err := json.Unmarshal(payloadBytes, &httpResp); err != nil {
			http.Error(w, "Invalid response payload", http.StatusBadGateway)
			return
		}

		// Write response headers
		for key, value := range httpResp.Headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(httpResp.StatusCode)
		if len(httpResp.Body) > 0 {
			w.Write(httpResp.Body)
		}

	case <-time.After(5 * time.Minute):
		http.Error(w, "Tunnel timeout", http.StatusGatewayTimeout)
	}
}

// isValidSubdomain checks if a subdomain name is valid
func isValidSubdomain(s string) bool {
	if len(s) < 1 || len(s) > 63 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return s[0] != '-' && s[len(s)-1] != '-'
}

// Run starts the server on the specified port
func (s *Server) Run(port int) error {
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting digit-link server on %s (domain: %s)", addr, s.domain)

	// Start ping routine
	go s.pingRoutine()

	return http.ListenAndServe(addr, s)
}

// pingRoutine sends periodic pings to keep connections alive
func (s *Server) pingRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		tunnels := make([]*Tunnel, 0, len(s.tunnels))
		for _, t := range s.tunnels {
			tunnels = append(tunnels, t)
		}
		s.mu.RUnlock()

		pingMsg, _ := json.Marshal(protocol.Message{Type: protocol.TypePing})
		for _, tunnel := range tunnels {
			tunnel.Conn.WriteMessage(websocket.TextMessage, pingMsg)
		}
	}
}

// GetDomain returns the server domain from environment or default
func GetDomain() string {
	if domain := os.Getenv("DOMAIN"); domain != "" {
		return domain
	}
	return "tunnel.digit.zone"
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
