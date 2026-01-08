package client

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/niekvdm/digit-link/internal/tunnel"
)

// TCPClient represents a TCP/yamux-based tunnel client with multi-forward support
type TCPClient struct {
	server    string
	token     string
	forwards  []tunnel.ForwardConfig
	insecure  bool
	session   *tunnel.Session
	tunnels   []tunnel.TunnelInfo
	connected bool
	mu        sync.RWMutex
	done      chan struct{}

	// Reconnection settings
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration

	// Proxy instances for each forward
	proxies map[string]*Proxy // subdomain -> proxy

	// Display
	model  *Model
}

// TCPConfig holds TCP client configuration
type TCPConfig struct {
	Server         string
	Token          string
	Forwards       []tunnel.ForwardConfig
	Insecure       bool // Skip TLS verification
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Timeout        time.Duration // Request timeout for proxies
}

// NewTCPClient creates a new TCP/yamux tunnel client
func NewTCPClient(cfg TCPConfig) *TCPClient {
	// Set defaults
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = -1 // Infinite retries
	}
	if cfg.InitialBackoff == 0 {
		cfg.InitialBackoff = 1 * time.Second
	}
	if cfg.MaxBackoff == 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	// Create proxy for each forward
	proxies := make(map[string]*Proxy)
	for _, fwd := range cfg.Forwards {
		proxies[fwd.Subdomain] = NewProxyWithTimeout("localhost", fwd.LocalPort, fwd.LocalHTTPS, cfg.Timeout)
	}

	return &TCPClient{
		server:         cfg.Server,
		token:          cfg.Token,
		forwards:       cfg.Forwards,
		insecure:       cfg.Insecure,
		done:           make(chan struct{}),
		maxRetries:     cfg.MaxRetries,
		initialBackoff: cfg.InitialBackoff,
		maxBackoff:     cfg.MaxBackoff,
		proxies:        proxies,
	}
}

// Connect establishes a TCP+TLS connection and authenticates
func (c *TCPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Parse server address
	address := c.server
	if !strings.Contains(address, ":") {
		address = address + ":4443" // Default tunnel port
	}

	// Establish TCP+TLS connection
	tlsConfig := tunnel.TLSClientConfig(c.getServerName(), c.insecure)
	conn, err := tunnel.DialTLS(address, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Enable TCP_NODELAY
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	// Create yamux client session
	session, err := tunnel.NewClientSession(conn, nil)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Open auth stream
	stream, err := session.Open()
	if err != nil {
		session.Close()
		return fmt.Errorf("failed to open auth stream: %w", err)
	}

	// Send auth request
	authReq := tunnel.AuthRequest{
		Token:    c.token,
		Forwards: c.forwards,
	}

	if err := tunnel.WriteFrame(stream, &authReq); err != nil {
		stream.Close()
		session.Close()
		return fmt.Errorf("failed to send auth request: %w", err)
	}

	// Read auth response
	stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	authResp, err := tunnel.ReadFrame[tunnel.AuthResponse](stream)
	if err != nil {
		stream.Close()
		session.Close()
		return fmt.Errorf("failed to read auth response: %w", err)
	}
	stream.Close()

	if !authResp.Success {
		session.Close()
		return fmt.Errorf("registration failed: %s", authResp.Error)
	}

	// Store session and tunnel info
	c.session = session
	c.tunnels = authResp.Tunnels
	c.connected = true

	// Copy LocalHTTPS from forwards to tunnels (matched by subdomain)
	for i := range c.tunnels {
		for _, fwd := range c.forwards {
			if fwd.Subdomain == c.tunnels[i].Subdomain {
				c.tunnels[i].LocalHTTPS = fwd.LocalHTTPS
				break
			}
		}
	}

	// Update session with forwards
	session.SetForwards(c.forwards)

	return nil
}

// Run starts the client with auto-reconnect
func (c *TCPClient) Run() error {
	retries := 0
	backoff := c.initialBackoff

	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		if err := c.Connect(); err != nil {
			errMsg := err.Error()

			// Check if fatal error
			if isFatalError(errMsg) {
				if c.model != nil {
					c.model.SendUpdate(StatusUpdateMsg{
						Status: "rejected",
						Server: c.server,
						Error:  extractErrorReason(errMsg),
					})
				}
				<-c.done
				return err
			}

			retries++
			if c.maxRetries > 0 && retries > c.maxRetries {
				return fmt.Errorf("max retries exceeded: %w", err)
			}

			if c.model != nil {
				c.model.SendUpdate(StatusUpdateMsg{
					Status: "connecting",
					Server: c.server,
				})
			}
			time.Sleep(backoff)
			backoff = backoff * 2
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
			continue
		}

		// Reset backoff
		retries = 0
		backoff = c.initialBackoff

		// Notify connected
		if c.model != nil {
			// Use first tunnel URL as public URL for display
			publicURL := ""
			if len(c.tunnels) > 0 {
				publicURL = c.tunnels[0].URL
			}
			c.model.SendUpdate(StatusUpdateMsg{
				Status:    "online",
				Server:    c.server,
				PublicURL: publicURL,
				Tunnels:   c.tunnels,
			})
		}

		// Handle incoming streams
		c.handleStreams()

		// Cleanup on disconnect
		c.mu.Lock()
		c.connected = false
		if c.session != nil {
			c.session.Close()
			c.session = nil
		}
		c.mu.Unlock()

		if c.model != nil {
			c.model.SendUpdate(StatusUpdateMsg{
				Status:       "reconnecting",
				Server:       c.server,
				RetryCount:   retries + 1,
				RetryBackoff: backoff,
			})
		}
	}
}

// handleStreams accepts and processes incoming yamux streams
func (c *TCPClient) handleStreams() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.mu.RLock()
		session := c.session
		c.mu.RUnlock()

		if session == nil || session.IsClosed() {
			return
		}

		// Accept incoming stream from server
		stream, err := session.AcceptStream()
		if err != nil {
			if session.IsClosed() {
				return
			}
			continue
		}

		// Handle request in goroutine
		go c.handleRequest(stream)
	}
}

// handleRequest processes a single HTTP request from a yamux stream
func (c *TCPClient) handleRequest(stream net.Conn) {
	defer stream.Close()

	startTime := time.Now()

	// Read request frame
	reqFrame, err := tunnel.ReadFrame[tunnel.RequestFrame](stream)
	if err != nil {
		return
	}

	// Find the proxy for this subdomain
	proxy, ok := c.proxies[reqFrame.Subdomain]
	if !ok {
		// Fallback to first proxy if subdomain not found
		for _, p := range c.proxies {
			proxy = p
			break
		}
	}

	if proxy == nil {
		// Send error response
		tunnel.WriteFrame(stream, &tunnel.ResponseFrame{
			ID:     reqFrame.ID,
			Status: 502,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: []byte("No proxy configured for subdomain"),
		})
		return
	}

	// Calculate bytes received
	bytesRecv := int64(len(reqFrame.Body))

	// Notify model of pending request
	if c.model != nil {
		c.model.SendUpdate(RequestAddedMsg{
			ID:        reqFrame.ID,
			Method:    reqFrame.Method,
			Path:      reqFrame.Path,
			Subdomain: reqFrame.Subdomain,
			BytesRecv: bytesRecv,
		})
	}

	// Forward to local service using existing proxy
	httpReq := &struct {
		ID      string
		Method  string
		Path    string
		Headers map[string]string
		Body    []byte
	}{
		ID:      reqFrame.ID,
		Method:  reqFrame.Method,
		Path:    reqFrame.Path,
		Headers: reqFrame.Headers,
		Body:    reqFrame.Body,
	}

	httpResp, err := proxy.ForwardRaw(httpReq.Method, httpReq.Path, httpReq.Headers, httpReq.Body)
	if err != nil {
		httpResp = &tunnel.ResponseFrame{
			ID:     reqFrame.ID,
			Status: 502,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: []byte(fmt.Sprintf("Proxy error: %v", err)),
		}
	} else {
		httpResp.ID = reqFrame.ID
	}

	duration := time.Since(startTime)
	bytesSent := int64(len(httpResp.Body))

	// Notify model of completed request
	if c.model != nil {
		c.model.SendUpdate(RequestCompletedMsg{
			ID:         reqFrame.ID,
			StatusCode: httpResp.Status,
			Duration:   duration,
			BytesSent:  bytesSent,
			BytesRecv:  bytesRecv,
		})
	}

	// Send response frame
	tunnel.WriteFrame(stream, httpResp)
}

// getServerName extracts the server name for TLS
func (c *TCPClient) getServerName() string {
	host := c.server
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// Close closes the client
func (c *TCPClient) Close() {
	close(c.done)
	c.mu.Lock()
	if c.session != nil {
		c.session.Close()
	}
	c.mu.Unlock()
}

// Tunnels returns the registered tunnel information
func (c *TCPClient) Tunnels() []tunnel.TunnelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tunnels
}

// IsConnected returns whether the client is connected
func (c *TCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SetModel sets the TUI model for status updates
func (c *TCPClient) SetModel(model *Model) {
	c.model = model
}
