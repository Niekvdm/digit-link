package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/niekvdm/digit-link/internal/protocol"
)

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

			log.Printf("Connection failed: %v (retry %d in %v)", err, retries, backoff)
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

		log.Printf("Connected! Public URL: %s", c.publicURL)
		log.Printf("Forwarding to localhost:%d", c.localPort)

		// Handle messages until disconnection
		c.handleMessages()

		c.mu.Lock()
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()

		log.Println("Disconnected, reconnecting...")
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
				log.Printf("Read error: %v", err)
			}
			return
		}

		var message protocol.Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Invalid message: %v", err)
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
	// Parse request payload
	payloadBytes, _ := json.Marshal(payload)
	var httpReq protocol.HTTPRequest
	if err := json.Unmarshal(payloadBytes, &httpReq); err != nil {
		log.Printf("Invalid HTTP request payload: %v", err)
		return
	}

	// Forward to local service
	httpResp, err := c.proxy.Forward(&httpReq)
	if err != nil {
		log.Printf("Forward error: %v", err)
		httpResp = ForwardError(httpReq.ID, 502, err.Error())
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
