// Package tunnel provides the TCP+TLS tunnel protocol types and session management
// using yamux for stream multiplexing over a single connection.
package tunnel

import (
	"encoding/json"
	"fmt"
	"io"
)

// Message types for TCP tunnel communication
const (
	TypeAuthRequest  = "auth_request"
	TypeAuthResponse = "auth_response"
	TypeHTTPRequest  = "http_request"
	TypeHTTPResponse = "http_response"
	TypePing         = "ping"
	TypePong         = "pong"
)

// ForwardConfig defines a single port forwarding configuration
type ForwardConfig struct {
	Subdomain  string `json:"subdomain"`
	LocalPort  int    `json:"localPort"`
	LocalHTTPS bool   `json:"localHttps,omitempty"` // Use HTTPS for local forwarding
	Primary    bool   `json:"primary,omitempty"`
}

// AuthRequest is sent by the client after establishing the yamux session
// to authenticate and register multiple forwards
type AuthRequest struct {
	Token    string          `json:"token"`
	Forwards []ForwardConfig `json:"forwards"`
	AppID    string          `json:"appId,omitempty"` // App ID when using app-specific API key
}

// TunnelInfo contains information about a registered tunnel endpoint
type TunnelInfo struct {
	Subdomain  string `json:"subdomain"`
	URL        string `json:"url"`
	LocalPort  int    `json:"localPort"`
	LocalHTTPS bool   `json:"-"` // Client-side only: forward to HTTPS locally
}

// AuthResponse is sent by the server to confirm or reject authentication
type AuthResponse struct {
	Success bool         `json:"success"`
	Tunnels []TunnelInfo `json:"tunnels,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// RequestFrame represents an HTTP request sent from server to client over a yamux stream
type RequestFrame struct {
	ID        string            `json:"id"`
	Subdomain string            `json:"subdomain"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      []byte            `json:"body,omitempty"`
}

// ResponseFrame represents an HTTP response sent from client to server over a yamux stream
type ResponseFrame struct {
	ID      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body,omitempty"`
}

// PingFrame is used for keepalive
type PingFrame struct {
	Timestamp int64 `json:"timestamp"`
}

// PongFrame is the response to a ping
type PongFrame struct {
	Timestamp int64 `json:"timestamp"`
}

// ReadFrame reads a JSON-encoded frame from a reader (yamux stream)
func ReadFrame[T any](r io.Reader) (*T, error) {
	decoder := json.NewDecoder(r)
	var frame T
	if err := decoder.Decode(&frame); err != nil {
		return nil, fmt.Errorf("failed to decode frame: %w", err)
	}
	return &frame, nil
}

// WriteFrame writes a JSON-encoded frame to a writer (yamux stream)
func WriteFrame[T any](w io.Writer, frame *T) error {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(frame); err != nil {
		return fmt.Errorf("failed to encode frame: %w", err)
	}
	return nil
}

// Validate checks if the AuthRequest has valid configuration
func (a *AuthRequest) Validate() error {
	if a.Token == "" {
		return fmt.Errorf("token is required")
	}
	if len(a.Forwards) == 0 {
		return fmt.Errorf("at least one forward is required")
	}

	subdomains := make(map[string]bool)
	primaryCount := 0
	for i, f := range a.Forwards {
		if f.Subdomain == "" {
			return fmt.Errorf("forward %d: subdomain is required", i)
		}
		if f.LocalPort <= 0 || f.LocalPort > 65535 {
			return fmt.Errorf("forward %d: invalid port %d", i, f.LocalPort)
		}
		if subdomains[f.Subdomain] {
			return fmt.Errorf("forward %d: duplicate subdomain %s", i, f.Subdomain)
		}
		subdomains[f.Subdomain] = true
		if f.Primary {
			primaryCount++
		}
	}

	if primaryCount > 1 {
		return fmt.Errorf("only one forward can be marked as primary")
	}

	return nil
}
