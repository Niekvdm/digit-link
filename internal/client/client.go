package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/niekvdm/digit-link/internal/protocol"
)

// RequestLog represents a logged request
type RequestLog struct {
	ID         string
	Time       time.Time
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	Pending    bool
}

// formatBytes formats bytes to human readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatUptime formats duration as uptime string
func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// Client represents a tunnel client
type Client struct {
	serverURL string
	subdomain string
	token     string
	secret    string // Legacy
	localPort int
	conn      *websocket.Conn
	proxy     *Proxy
	publicURL string
	connected bool
	mu        sync.RWMutex
	done      chan struct{}

	// Reconnection settings
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration

	// Display
	model  *Model
	server string // Original server hostname for display
}

// Config holds client configuration
type Config struct {
	Server         string
	Subdomain      string
	Token          string
	Secret         string // Legacy support
	LocalPort      int
	Timeout        time.Duration // Request timeout (default: 5 minutes)
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Insecure       bool // Use ws:// instead of wss://
}

// New creates a new tunnel client
func New(cfg Config) *Client {
	// Build WebSocket URL
	scheme := "wss"
	if cfg.Insecure {
		scheme = "ws"
	}
	wsURL := fmt.Sprintf("%s://%s/_tunnel", scheme, cfg.Server)

	// Set defaults for reconnection
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = -1 // Infinite retries by default
	}
	if cfg.InitialBackoff == 0 {
		cfg.InitialBackoff = 1 * time.Second
	}
	if cfg.MaxBackoff == 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute // Default 5 minute timeout
	}

	c := &Client{
		serverURL:      wsURL,
		subdomain:      cfg.Subdomain,
		token:          cfg.Token,
		secret:         cfg.Secret,
		localPort:      cfg.LocalPort,
		proxy:          NewProxyWithTimeout(cfg.LocalPort, cfg.Timeout),
		done:           make(chan struct{}),
		maxRetries:     cfg.MaxRetries,
		initialBackoff: cfg.InitialBackoff,
		maxBackoff:     cfg.MaxBackoff,
		server:         cfg.Server,
	}
	c.model = NewModel(c, cfg.Server, cfg.LocalPort)
	return c
}

// Connect establishes a connection to the tunnel server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Parse URL for dialer
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// Connect with WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn

	// Send registration request
	regReq := protocol.Message{
		Type: protocol.TypeRegisterRequest,
		Payload: protocol.RegisterRequest{
			Subdomain: c.subdomain,
			Token:     c.token,
			Secret:    c.secret, // Legacy support
		},
	}

	data, _ := json.Marshal(regReq)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send registration: %w", err)
	}

	// Wait for response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to read registration response: %w", err)
	}
	conn.SetReadDeadline(time.Time{})

	var respMsg protocol.Message
	if err := json.Unmarshal(msg, &respMsg); err != nil {
		conn.Close()
		return fmt.Errorf("invalid registration response: %w", err)
	}

	// Parse response payload
	payloadBytes, _ := json.Marshal(respMsg.Payload)
	var regResp protocol.RegisterResponse
	if err := json.Unmarshal(payloadBytes, &regResp); err != nil {
		conn.Close()
		return fmt.Errorf("invalid registration response payload: %w", err)
	}

	if !regResp.Success {
		conn.Close()
		return fmt.Errorf("registration failed: %s", regResp.Error)
	}

	c.publicURL = regResp.URL
	c.connected = true

	return nil
}

// Run starts the client message loop with auto-reconnect
func (c *Client) Run() error {
	retries := 0
	backoff := c.initialBackoff

	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		// Connect if not connected
		if err := c.Connect(); err != nil {
			errMsg := err.Error()

			// Check if this is a fatal (non-retriable) error
			if isFatalError(errMsg) {
				if c.model != nil {
					c.model.SendUpdate(StatusUpdateMsg{
						Status: "rejected",
						Server: c.server,
						Error:  extractErrorReason(errMsg),
					})
				}
				// For fatal errors, don't retry - just wait for quit
				// This keeps the UI visible so user can see the error
				<-c.done
				return err
			}

			retries++
			if c.maxRetries > 0 && retries > c.maxRetries {
				return fmt.Errorf("max retries exceeded: %w", err)
			}

			// Update model to show connecting status
			if c.model != nil {
				c.model.SendUpdate(StatusUpdateMsg{
					Status:    "connecting",
					Server:    c.server,
					PublicURL: "",
				})
			}
			time.Sleep(backoff)

			// Exponential backoff
			backoff = backoff * 2
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
			continue
		}

		// Reset backoff on successful connection
		retries = 0
		backoff = c.initialBackoff

		// Update model with initial status
		if c.model != nil {
			c.model.SendUpdate(StatusUpdateMsg{
				Status:    "online",
				Server:    c.server,
				PublicURL: c.publicURL,
			})
		}

		// Handle messages until disconnection
		c.handleMessages()

		c.mu.Lock()
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		// Update model to show reconnecting status
		if c.model != nil {
			c.model.SendUpdate(StatusUpdateMsg{
				Status:    "reconnecting",
				Server:    c.server,
				PublicURL: c.publicURL,
			})
		}
	}
}

// handleMessages processes incoming messages from the server
func (c *Client) handleMessages() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				// Connection error - will reconnect
			}
			return
		}

		var message protocol.Message
		if err := json.Unmarshal(msg, &message); err != nil {
			continue
		}

		switch message.Type {
		case protocol.TypeHTTPRequest:
			go c.handleHTTPRequest(message.Payload)
		case protocol.TypePing:
			c.sendPong()
		}
	}
}

// handleHTTPRequest handles an incoming HTTP request from the tunnel
func (c *Client) handleHTTPRequest(payload interface{}) {
	startTime := time.Now()

	// Parse request payload
	payloadBytes, _ := json.Marshal(payload)
	var httpReq protocol.HTTPRequest
	if err := json.Unmarshal(payloadBytes, &httpReq); err != nil {
		return
	}

	// Add request immediately as pending
	if c.model != nil {
		c.model.SendUpdate(RequestAddedMsg{
			ID:     httpReq.ID,
			Method: httpReq.Method,
			Path:   httpReq.Path,
		})
	}

	// Calculate bytes received (request body)
	bytesRecv := int64(len(httpReq.Body))

	// Forward to local service
	httpResp, err := c.proxy.Forward(&httpReq)
	if err != nil {
		httpResp = ForwardError(httpReq.ID, 502, err.Error())
	}

	duration := time.Since(startTime)

	// Calculate bytes sent (response body)
	bytesSent := int64(len(httpResp.Body))

	// Mark request as complete
	if c.model != nil {
		c.model.SendUpdate(RequestCompletedMsg{
			ID:         httpReq.ID,
			StatusCode: httpResp.StatusCode,
			Duration:   duration,
			BytesSent:  bytesSent,
			BytesRecv:  bytesRecv,
		})
	}

	// Send response back
	respMsg := protocol.Message{
		Type:    protocol.TypeHTTPResponse,
		Payload: httpResp,
	}

	data, _ := json.Marshal(respMsg)

	c.mu.RLock()
	if c.conn != nil {
		c.conn.WriteMessage(websocket.TextMessage, data)
	}
	c.mu.RUnlock()
}

// sendPong sends a pong response
func (c *Client) sendPong() {
	pongMsg, _ := json.Marshal(protocol.Message{Type: protocol.TypePong})
	c.mu.RLock()
	if c.conn != nil {
		c.conn.WriteMessage(websocket.TextMessage, pongMsg)
	}
	c.mu.RUnlock()
}


// Close closes the client connection
func (c *Client) Close() {
	close(c.done)
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

// PublicURL returns the public URL of the tunnel
func (c *Client) PublicURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.publicURL
}

// Model returns the Bubbletea model for UI rendering
func (c *Client) Model() *Model {
	return c.model
}

// isFatalError checks if the error is a fatal (non-retriable) error
// Fatal errors include authentication failures, quota exceeded, IP not whitelisted, etc.
func isFatalError(errMsg string) bool {
	fatalPatterns := []string{
		"Authentication required",
		"Invalid token",
		"API key",
		"not whitelisted",
		"IP address not whitelisted",
		"Quota exceeded",
		"Invalid subdomain",
		"Subdomain already in use",
		"Application not found",
		"expired",
	}

	errLower := strings.ToLower(errMsg)
	for _, pattern := range fatalPatterns {
		if strings.Contains(errLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// extractErrorReason extracts the user-friendly error reason from an error message
// It removes the "registration failed: " prefix if present
func extractErrorReason(errMsg string) string {
	// Remove common prefixes
	prefixes := []string{
		"registration failed: ",
		"failed to connect: ",
	}

	result := errMsg
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(result), strings.ToLower(prefix)) {
			result = result[len(prefix):]
		}
	}
	return result
}
