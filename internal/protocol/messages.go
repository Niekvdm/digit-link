package protocol

import "encoding/json"

// Message types for WebSocket communication between client and server
const (
	TypeRegisterRequest  = "register_request"
	TypeRegisterResponse = "register_response"
	TypeHTTPRequest      = "http_request"
	TypeHTTPResponse     = "http_response"
	TypePing             = "ping"
	TypePong             = "pong"
)

// Message is the base wrapper for all WebSocket messages
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// TypedMessage uses json.RawMessage to avoid double serialization
type TypedMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// RegisterRequest is sent by the client to register a subdomain
type RegisterRequest struct {
	Subdomain string `json:"subdomain"`
	Secret    string `json:"secret,omitempty"` // Deprecated: use Token instead
	Token     string `json:"token,omitempty"`  // Authentication token (account token or API key)
	AppID     string `json:"appId,omitempty"`  // App ID when using app-specific API key
}

// RegisterResponse is sent by the server to confirm or reject registration
type RegisterResponse struct {
	Success   bool   `json:"success"`
	Subdomain string `json:"subdomain,omitempty"`
	URL       string `json:"url,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HTTPRequest represents an incoming HTTP request to be forwarded
type HTTPRequest struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body,omitempty"`
}

// HTTPResponse represents the response from the local service
type HTTPResponse struct {
	ID         string            `json:"id"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body,omitempty"`
}
