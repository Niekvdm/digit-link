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

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m"
	colorYellow  = "\033[33m"
	colorGreen   = "\033[32m"
	colorMagenta = "\033[35m"
	colorWhite   = "\033[37m"
	colorGray    = "\033[90m"
	colorRed     = "\033[31m"
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

// Display handles the console UI
type Display struct {
	mu          sync.Mutex
	requests    []RequestLog
	maxRequests int
	bytesSent   int64
	bytesRecv   int64
	startTime   time.Time
	initialized bool
}

// NewDisplay creates a new display
func NewDisplay() *Display {
	return &Display{
		requests:    make([]RequestLog, 0, 5),
		maxRequests: 5,
		startTime:   time.Now(),
	}
}

// AddRequest adds a request to the log
func (d *Display) AddRequest(id, method, path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	req := RequestLog{
		ID:      id,
		Time:    time.Now(),
		Method:  method,
		Path:    path,
		Pending: true,
	}

	d.requests = append(d.requests, req)
	if len(d.requests) > d.maxRequests {
		d.requests = d.requests[1:]
	}
}

// CompleteRequest marks a request as complete
func (d *Display) CompleteRequest(id string, statusCode int, duration time.Duration, bytesSent, bytesRecv int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := range d.requests {
		if d.requests[i].ID == id {
			d.requests[i].StatusCode = statusCode
			d.requests[i].Duration = duration
			d.requests[i].Pending = false
			break
		}
	}

	d.bytesSent += bytesSent
	d.bytesRecv += bytesRecv
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

// Render renders the display to the terminal
func (d *Display) Render(status, server, publicURL string, localPort int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var sb strings.Builder

	// Move cursor to home position and clear screen (only on first render)
	if !d.initialized {
		sb.WriteString("\033[2J") // Clear screen
		d.initialized = true
	}
	sb.WriteString("\033[H") // Move to home

	// Header
	sb.WriteString(fmt.Sprintf("%sdigit-link%s                                                    %s(Ctrl+C to quit)%s\n", colorCyan, colorReset, colorGray, colorReset))
	sb.WriteString("\n")

	// Status section
	statusColor := colorGreen
	if status != "online" {
		statusColor = colorYellow
	}
	sb.WriteString(fmt.Sprintf("%-20s %s%s%s\n", colorYellow+"Session Status"+colorReset, statusColor, status, colorReset))
	sb.WriteString(fmt.Sprintf("%-20s %s\n", colorYellow+"Version"+colorReset, "1.0.0"))
	sb.WriteString(fmt.Sprintf("%-20s %s\n", colorYellow+"Server"+colorReset, server))
	sb.WriteString(fmt.Sprintf("%-20s %s\n", colorYellow+"Forwarding"+colorReset, fmt.Sprintf("%s%s%s -> %shttp://localhost:%d%s", colorMagenta, publicURL, colorReset, colorWhite, localPort, colorReset)))
	sb.WriteString("\n")

	// Stats section
	uptime := time.Since(d.startTime)
	sb.WriteString(fmt.Sprintf("%sStats%s          %-15s %-15s %-15s\n", colorYellow, colorReset, "Uptime", "Sent", "Received"))
	sb.WriteString(fmt.Sprintf("               %-15s %-15s %-15s\n", formatUptime(uptime), formatBytes(d.bytesSent), formatBytes(d.bytesRecv)))
	sb.WriteString("\n")

	// Recent requests header
	sb.WriteString(fmt.Sprintf("%sRecent Requests%s\n", colorYellow, colorReset))
	sb.WriteString(fmt.Sprintf("%s─────────────────────────────────────────────────────────────────────────────────%s\n", colorGray, colorReset))

	// Display last 5 requests (or empty lines)
	for i := 0; i < d.maxRequests; i++ {
		if i < len(d.requests) {
			req := d.requests[len(d.requests)-1-i] // Most recent first

			// Truncate path if too long
			path := req.Path
			if len(path) > 40 {
				path = path[:37] + "..."
			}

			if req.Pending {
				// Show pending request with spinner
				spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
				spinIdx := int(time.Since(req.Time).Milliseconds()/100) % len(spinChars)
				sb.WriteString(fmt.Sprintf("%s%-8s%s %-7s %-40s %s%s%s %8s\n",
					colorGray,
					req.Time.Format("15:04:05"),
					colorReset,
					req.Method,
					path,
					colorYellow,
					spinChars[spinIdx],
					colorReset,
					"...",
				))
			} else {
				statusColor := colorGreen
				if req.StatusCode >= 400 && req.StatusCode < 500 {
					statusColor = colorYellow
				} else if req.StatusCode >= 500 {
					statusColor = colorRed
				}

				sb.WriteString(fmt.Sprintf("%s%-8s%s %-7s %-40s %s%3d%s %8s\n",
					colorGray,
					req.Time.Format("15:04:05"),
					colorReset,
					req.Method,
					path,
					statusColor,
					req.StatusCode,
					colorReset,
					formatDuration(req.Duration),
				))
			}
		} else {
			// Empty line placeholder (clears previous content)
			sb.WriteString(fmt.Sprintf("%-80s\n", ""))
		}
	}

	fmt.Print(sb.String())
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
	display       *Display
	server        string // Original server hostname for display
	displayTicker *time.Ticker
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

	return &Client{
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
		display:        NewDisplay(),
		server:         cfg.Server,
	}
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
			retries++
			if c.maxRetries > 0 && retries > c.maxRetries {
				return fmt.Errorf("max retries exceeded: %w", err)
			}

			// Update display to show connecting status
			c.display.Render("connecting", c.server, "", c.localPort)
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

		// Render initial display
		c.display.Render("online", c.server, c.publicURL, c.localPort)

		// Start display refresh ticker
		c.displayTicker = time.NewTicker(1 * time.Second)
		go c.refreshDisplay()

		// Handle messages until disconnection
		c.handleMessages()

		// Stop display ticker
		c.displayTicker.Stop()

		c.mu.Lock()
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		// Update display to show reconnecting status
		c.display.Render("reconnecting", c.server, c.publicURL, c.localPort)
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
	c.display.AddRequest(httpReq.ID, httpReq.Method, httpReq.Path)
	c.display.Render("online", c.server, c.publicURL, c.localPort)

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
	c.display.CompleteRequest(httpReq.ID, httpResp.StatusCode, duration, bytesSent, bytesRecv)

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

	// Update display
	c.display.Render("online", c.server, c.publicURL, c.localPort)
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

// refreshDisplay periodically refreshes the display
func (c *Client) refreshDisplay() {
	for {
		select {
		case <-c.done:
			return
		case <-c.displayTicker.C:
			c.mu.RLock()
			connected := c.connected
			publicURL := c.publicURL
			c.mu.RUnlock()

			if connected {
				c.display.Render("online", c.server, publicURL, c.localPort)
			}
		}
	}
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
