package server

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/tunnel"
	proxyproto "github.com/pires/go-proxyproto"
)

// TunnelListener handles TCP+TLS tunnel connections using yamux
type TunnelListener struct {
	server    *Server
	listener  net.Listener
	tlsConfig *tls.Config
	sessions  map[string]*tunnel.Session // subdomain -> session
	mu        sync.RWMutex
	done      chan struct{}
}

// NewTunnelListener creates a new TCP tunnel listener
func NewTunnelListener(server *Server, tlsConfig *tls.Config) *TunnelListener {
	return &TunnelListener{
		server:    server,
		tlsConfig: tlsConfig,
		sessions:  make(map[string]*tunnel.Session),
		done:      make(chan struct{}),
	}
}

// Start begins listening for TCP+TLS connections on the specified port
func (tl *TunnelListener) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)

	var listener net.Listener
	var err error

	if tl.tlsConfig != nil {
		listener, err = tls.Listen("tcp", addr, tl.tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to start TLS listener: %w", err)
		}
		log.Printf("TCP+TLS tunnel listener started on %s", addr)
	} else {
		// Non-TLS mode for development/testing
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to start TCP listener: %w", err)
		}
		log.Printf("TCP tunnel listener started on %s (WARNING: TLS disabled)", addr)
	}

	// Wrap listener with PROXY protocol support to get real client IPs
	// when behind HAProxy or other load balancers using PROXY protocol
	proxyListener := &proxyproto.Listener{
		Listener: listener,
		Policy: func(upstream net.Addr) (proxyproto.Policy, error) {
			// Accept PROXY protocol from any source (trusted internal network)
			// The header is optional - connections without it still work
			return proxyproto.USE, nil
		},
	}
	tl.listener = proxyListener

	go tl.acceptLoop()

	return nil
}

// acceptLoop accepts incoming TCP connections
func (tl *TunnelListener) acceptLoop() {
	for {
		select {
		case <-tl.done:
			return
		default:
		}

		conn, err := tl.listener.Accept()
		if err != nil {
			select {
			case <-tl.done:
				return
			default:
				log.Printf("TCP accept error: %v", err)
				continue
			}
		}

		go tl.handleConnection(conn)
	}
}

// handleConnection handles a new TCP connection
func (tl *TunnelListener) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New TCP tunnel connection from %s", remoteAddr)

	// Enable TCP_NODELAY for lower latency
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	} else if tlsConn, ok := conn.(*tls.Conn); ok {
		// For TLS connections, get the underlying TCP connection
		if tcpConn, ok := tlsConn.NetConn().(*net.TCPConn); ok {
			tcpConn.SetNoDelay(true)
		}
	}

	// Create yamux server session
	session, err := tunnel.NewServerSession(conn, nil)
	if err != nil {
		log.Printf("Failed to create yamux session for %s: %v", remoteAddr, err)
		conn.Close()
		return
	}

	// Handle the session (auth, registration, etc.)
	// This will be implemented in task 399
	tl.handleSession(session)
}

// handleSession handles a yamux session after establishment
func (tl *TunnelListener) handleSession(session *tunnel.Session) {
	remoteAddr := session.RemoteAddr().String()
	clientIP := extractIPFromAddr(remoteAddr)
	log.Printf("Yamux session established with %s", remoteAddr)

	// Accept the first stream for authentication
	stream, err := session.AcceptStream()
	if err != nil {
		log.Printf("Failed to accept auth stream from %s: %v", remoteAddr, err)
		session.Close()
		return
	}

	// Read auth request
	authReq, err := tunnel.ReadFrame[tunnel.AuthRequest](stream)
	if err != nil {
		log.Printf("Failed to read auth request from %s: %v", remoteAddr, err)
		stream.Close()
		session.Close()
		return
	}

	// Validate auth request structure
	if err := authReq.Validate(); err != nil {
		log.Printf("Invalid auth request from %s: %v", remoteAddr, err)
		tunnel.WriteFrame(stream, &tunnel.AuthResponse{
			Success: false,
			Error:   err.Error(),
		})
		stream.Close()
		session.Close()
		return
	}

	// Authenticate and register the session
	authResult := tl.authenticateSession(session, authReq, clientIP)

	// Send auth response
	if err := tunnel.WriteFrame(stream, authResult.response); err != nil {
		log.Printf("Failed to send auth response to %s: %v", remoteAddr, err)
		stream.Close()
		session.Close()
		return
	}
	stream.Close()

	if !authResult.response.Success {
		log.Printf("Authentication failed for %s: %s", remoteAddr, authResult.response.Error)
		session.Close()
		return
	}

	// Register session with all subdomains
	session.SetForwards(authReq.Forwards)
	session.SetAccountInfo(authResult.accountID, authResult.orgID, authResult.appID)

	if err := tl.RegisterSession(session); err != nil {
		log.Printf("Failed to register session from %s: %v", remoteAddr, err)
		session.Close()
		return
	}

	// Log successful registration
	for _, t := range authResult.response.Tunnels {
		log.Printf("TCP tunnel registered: %s -> %s (ip: %s)", t.Subdomain, t.URL, clientIP)
	}

	// Track session start time for usage
	sessionStartTime := time.Now()

	// Increment concurrent tunnels for quota tracking
	if tl.server.usageCache != nil && authResult.orgID != "" {
		tl.server.usageCache.IncrementConcurrentTunnels(authResult.orgID)
	}

	// Keep session alive - this blocks until session closes
	tl.maintainSession(session)

	// Cleanup on disconnect
	tl.UnregisterSession(session)
	session.Close()

	// Track usage on disconnect
	if tl.server.usageCache != nil && authResult.orgID != "" {
		tl.server.usageCache.DecrementConcurrentTunnels(authResult.orgID)
		sessionDuration := time.Since(sessionStartTime)
		tl.server.usageCache.RecordTunnelTime(authResult.orgID, int64(sessionDuration.Seconds()))
	}

	log.Printf("TCP tunnel session disconnected: %s", remoteAddr)
}

// authResult holds the result of authentication
type authResult struct {
	response  *tunnel.AuthResponse
	accountID string
	orgID     string
	appID     string
}

// authenticateSession validates the auth request and returns the result
func (tl *TunnelListener) authenticateSession(session *tunnel.Session, authReq *tunnel.AuthRequest, clientIP string) *authResult {
	result := &authResult{
		response: &tunnel.AuthResponse{Success: false},
	}

	if tl.server.db == nil {
		result.response.Error = "Database not configured"
		return result
	}

	// Try API key authentication first
	apiKeyHash := db.HashAPIKey(authReq.Token)
	apiKey, err := tl.server.db.GetAPIKeyByHash(apiKeyHash)
	if err != nil {
		log.Printf("Database error during API key lookup: %v", err)
		result.response.Error = "Internal server error"
		return result
	}

	var account *db.Account

	if apiKey != nil {
		// API key authentication
		if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
			result.response.Error = "API key has expired"
			return result
		}

		if apiKey.KeyType == db.KeyTypeApp && apiKey.AppID != nil {
			// App-specific API key
			app, err := tl.server.db.GetApplicationByID(*apiKey.AppID)
			if err != nil || app == nil {
				result.response.Error = "Application not found for API key"
				return result
			}
			result.orgID = app.OrgID
			result.appID = app.ID

			// Check app-level IP whitelist
			whitelisted, err := tl.server.db.IsIPWhitelistedForApp(clientIP, app.ID)
			if err != nil {
				result.response.Error = "Internal server error"
				return result
			}
			if !whitelisted {
				result.response.Error = "IP address not whitelisted"
				return result
			}
		} else if apiKey.OrgID != nil {
			// Org-level API key
			result.orgID = *apiKey.OrgID

			// Check org-level IP whitelist
			whitelisted, err := tl.server.db.IsIPWhitelistedForOrg(clientIP, result.orgID)
			if err != nil {
				result.response.Error = "Internal server error"
				return result
			}
			if !whitelisted {
				result.response.Error = "IP address not whitelisted"
				return result
			}
		}

		tl.server.db.UpdateAPIKeyLastUsed(apiKey.ID)
	} else {
		// Try account token authentication
		tokenHash := auth.HashToken(authReq.Token)
		account, err = tl.server.db.GetAccountByTokenHash(tokenHash)
		if err != nil {
			result.response.Error = "Internal server error"
			return result
		}
		if account == nil {
			result.response.Error = "Invalid token"
			return result
		}

		result.accountID = account.ID
		result.orgID = account.OrgID

		// Check account IP whitelist
		whitelisted, err := tl.server.db.IsIPWhitelistedForAccount(clientIP, account.ID)
		if err != nil {
			result.response.Error = "Internal server error"
			return result
		}
		if !whitelisted {
			result.response.Error = "IP address not whitelisted"
			return result
		}

		tl.server.db.UpdateAccountLastUsed(account.ID)
	}

	// Validate and register subdomains
	tunnels := make([]tunnel.TunnelInfo, 0, len(authReq.Forwards))
	for _, fwd := range authReq.Forwards {
		subdomain := strings.ToLower(fwd.Subdomain)

		// Validate subdomain
		if !isValidSubdomain(subdomain) {
			result.response.Error = fmt.Sprintf("Invalid subdomain: %s", subdomain)
			return result
		}

		// Check if subdomain is already in use (WebSocket tunnels)
		tl.server.mu.RLock()
		_, wsExists := tl.server.tunnels[subdomain]
		tl.server.mu.RUnlock()
		if wsExists {
			result.response.Error = fmt.Sprintf("Subdomain %s already in use", subdomain)
			return result
		}

		// Check if subdomain is already in use (TCP tunnels)
		tl.mu.RLock()
		_, tcpExists := tl.sessions[subdomain]
		tl.mu.RUnlock()
		if tcpExists {
			result.response.Error = fmt.Sprintf("Subdomain %s already in use", subdomain)
			return result
		}

		// Check quota before registering
		if tl.server.quotaChecker != nil && result.orgID != "" {
			allowed, reason := tl.server.quotaChecker.CanConnectTunnel(result.orgID)
			if !allowed {
				result.response.Error = fmt.Sprintf("Quota exceeded: %s", reason)
				return result
			}
		}

		url := fmt.Sprintf("%s://%s.%s", tl.server.scheme, subdomain, tl.server.domain)
		tunnels = append(tunnels, tunnel.TunnelInfo{
			Subdomain: subdomain,
			URL:       url,
			LocalPort: fwd.LocalPort,
		})
	}

	result.response.Success = true
	result.response.Tunnels = tunnels
	return result
}

// maintainSession keeps the session alive until it's closed
func (tl *TunnelListener) maintainSession(session *tunnel.Session) {
	// The session will be kept alive by yamux's built-in keepalive
	// We just need to wait for the session to be closed
	for {
		if session.IsClosed() {
			return
		}
		time.Sleep(time.Second)
	}
}

// extractIPFromAddr extracts the IP address from a remote address string
func extractIPFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// RegisterSession registers a session for multiple subdomains
func (tl *TunnelListener) RegisterSession(session *tunnel.Session) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	subdomains := session.GetSubdomains()
	for _, subdomain := range subdomains {
		if _, exists := tl.sessions[subdomain]; exists {
			return fmt.Errorf("subdomain %s already registered", subdomain)
		}
	}

	for _, subdomain := range subdomains {
		tl.sessions[subdomain] = session
	}

	return nil
}

// UnregisterSession removes a session's subdomains from the registry
func (tl *TunnelListener) UnregisterSession(session *tunnel.Session) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	for _, subdomain := range session.GetSubdomains() {
		delete(tl.sessions, subdomain)
	}
}

// GetSession returns the session for a subdomain
func (tl *TunnelListener) GetSession(subdomain string) (*tunnel.Session, bool) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	session, ok := tl.sessions[subdomain]
	return session, ok
}

// Stop gracefully stops the tunnel listener
func (tl *TunnelListener) Stop() error {
	close(tl.done)

	if tl.listener != nil {
		if err := tl.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	// Close all active sessions
	tl.mu.Lock()
	for subdomain, session := range tl.sessions {
		log.Printf("Closing TCP tunnel session for %s", subdomain)
		session.Close()
	}
	tl.sessions = make(map[string]*tunnel.Session)
	tl.mu.Unlock()

	return nil
}

// GetTunnelPort returns the TCP tunnel port from environment or default
func GetTunnelPort() int {
	if port := os.Getenv("TUNNEL_PORT"); port != "" {
		var p int
		fmt.Sscanf(port, "%d", &p)
		if p > 0 {
			return p
		}
	}
	return 4443
}

// GetTunnelTLSCertFile returns the TLS certificate file path from environment
func GetTunnelTLSCertFile() string {
	return os.Getenv("TUNNEL_TLS_CERT")
}

// GetTunnelTLSKeyFile returns the TLS key file path from environment
func GetTunnelTLSKeyFile() string {
	return os.Getenv("TUNNEL_TLS_KEY")
}

// IsTunnelEnabled returns true if the TCP tunnel listener should be enabled
func IsTunnelEnabled() bool {
	// Enable by default if TLS cert and key are provided, or if explicitly enabled
	if os.Getenv("TUNNEL_ENABLED") == "true" {
		return true
	}
	if os.Getenv("TUNNEL_ENABLED") == "false" {
		return false
	}
	// Auto-enable if TLS files are configured
	return GetTunnelTLSCertFile() != "" && GetTunnelTLSKeyFile() != ""
}
