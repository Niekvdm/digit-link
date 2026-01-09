package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/niekvdm/digit-link/internal/protocol"
	"github.com/niekvdm/digit-link/internal/tunnel"
)

// IsWebSocketUpgrade checks if the request headers indicate a WebSocket upgrade
func IsWebSocketUpgrade(headers map[string]string) bool {
	// Check for Connection: Upgrade (case-insensitive)
	connection := headers["Connection"]
	if connection == "" {
		connection = headers["connection"]
	}
	hasUpgradeConnection := strings.Contains(strings.ToLower(connection), "upgrade")

	// Check for Upgrade: websocket (case-insensitive)
	upgrade := headers["Upgrade"]
	if upgrade == "" {
		upgrade = headers["upgrade"]
	}
	hasWebSocketUpgrade := strings.EqualFold(upgrade, "websocket")

	return hasUpgradeConnection && hasWebSocketUpgrade
}

// Pipe copies data bidirectionally between two connections until one closes
// Returns the total bytes transferred in each direction
func Pipe(conn1, conn2 net.Conn) (int64, int64) {
	var wg sync.WaitGroup
	var bytesConn1ToConn2, bytesConn2ToConn1 int64

	wg.Add(2)

	// conn1 -> conn2
	go func() {
		defer wg.Done()
		bytesConn1ToConn2, _ = io.Copy(conn2, conn1)
		// Signal EOF to conn2 when conn1 is done reading
		if tcpConn, ok := conn2.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		} else if closeWriter, ok := conn2.(interface{ CloseWrite() error }); ok {
			closeWriter.CloseWrite()
		}
	}()

	// conn2 -> conn1
	go func() {
		defer wg.Done()
		bytesConn2ToConn1, _ = io.Copy(conn1, conn2)
		// Signal EOF to conn1 when conn2 is done reading
		if tcpConn, ok := conn1.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		} else if closeWriter, ok := conn1.(interface{ CloseWrite() error }); ok {
			closeWriter.CloseWrite()
		}
	}()

	wg.Wait()
	return bytesConn1ToConn2, bytesConn2ToConn1
}

// WebSocketUpgradeResult contains the result of a WebSocket upgrade attempt
type WebSocketUpgradeResult struct {
	Conn       net.Conn          // Raw connection to local service (if upgrade successful)
	StatusCode int               // HTTP status code from local service
	Headers    map[string]string // Response headers
	Success    bool              // Whether upgrade was successful (status 101)
}

// ForwardWebSocket attempts a WebSocket upgrade to the local service
// Returns the raw connection for bidirectional piping if successful
func (p *Proxy) ForwardWebSocket(method, path string, headers map[string]string, body []byte) (*WebSocketUpgradeResult, error) {
	// Parse local address to get host:port
	// p.localAddr is like "http://localhost:3000" or "https://localhost:3000"
	localURL := p.localAddr
	host := strings.TrimPrefix(localURL, "http://")
	host = strings.TrimPrefix(host, "https://")

	// Connect to local service
	conn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to local service: %w", err)
	}

	// Build HTTP upgrade request manually
	var reqBuf bytes.Buffer
	fmt.Fprintf(&reqBuf, "%s %s HTTP/1.1\r\n", method, path)

	// Write headers - include Connection and Upgrade for WebSocket
	for key, value := range headers {
		fmt.Fprintf(&reqBuf, "%s: %s\r\n", key, value)
	}
	reqBuf.WriteString("\r\n")

	// Write body if present
	if len(body) > 0 {
		reqBuf.Write(body)
	}

	// Send request
	if _, err := conn.Write(reqBuf.Bytes()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send upgrade request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read upgrade response: %w", err)
	}

	// Extract response headers
	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = values[0]
		}
	}

	result := &WebSocketUpgradeResult{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Success:    resp.StatusCode == http.StatusSwitchingProtocols,
	}

	if result.Success {
		// Upgrade successful - return the connection for piping
		result.Conn = conn
	} else {
		// Not a 101 response - close connection
		conn.Close()
	}

	return result, nil
}

// Proxy handles forwarding requests to the local service
type Proxy struct {
	localAddr string
	client    *http.Client
}

// DefaultTimeout is the default timeout for forwarding requests (5 minutes)
const DefaultTimeout = 5 * time.Minute

// NewProxy creates a new local proxy
func NewProxy(localAddr string, localPort int, useHTTPS bool) *Proxy {
	return NewProxyWithTimeout(localAddr, localPort, useHTTPS, DefaultTimeout)
}

// NewProxyWithTimeout creates a new local proxy with a custom timeout
func NewProxyWithTimeout(localAddr string, localPort int, useHTTPS bool, timeout time.Duration) *Proxy {
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}
	return &Proxy{
		localAddr: fmt.Sprintf("%s://%s:%d", scheme, localAddr, localPort),
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		},
	}
}

// Forward forwards an HTTP request to the local service and returns the response
func (p *Proxy) Forward(req *protocol.HTTPRequest) (*protocol.HTTPResponse, error) {
	// Build local request URL
	url := p.localAddr + req.Path

	// Create HTTP request
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, value := range req.Headers {
		// Skip hop-by-hop headers
		switch key {
		case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
			"Te", "Trailers", "Transfer-Encoding", "Upgrade":
			continue
		}
		httpReq.Header.Set(key, value)
	}

	// Execute request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Build response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		// Skip hop-by-hop headers
		switch key {
		case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
			"Te", "Trailers", "Transfer-Encoding", "Upgrade":
			continue
		}
		headers[key] = values[0]
	}

	return &protocol.HTTPResponse{
		ID:         req.ID,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       respBody,
	}, nil
}

// ForwardError creates an error response for failed requests
func ForwardError(requestID string, statusCode int, message string) *protocol.HTTPResponse {
	body, _ := json.Marshal(map[string]string{"error": message})
	return &protocol.HTTPResponse{
		ID:         requestID,
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

// ForwardRaw forwards a raw HTTP request and returns a tunnel.ResponseFrame
// Used by the TCP client for yamux-based forwarding
func (p *Proxy) ForwardRaw(method, path string, headers map[string]string, reqBody []byte) (*tunnel.ResponseFrame, error) {
	url := p.localAddr + path

	var body io.Reader
	if len(reqBody) > 0 {
		body = bytes.NewReader(reqBody)
	}

	httpReq, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		switch key {
		case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
			"Te", "Trailers", "Transfer-Encoding", "Upgrade":
			continue
		}
		httpReq.Header.Set(key, value)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		switch key {
		case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
			"Te", "Trailers", "Transfer-Encoding", "Upgrade":
			continue
		}
		respHeaders[key] = values[0]
	}

	return &tunnel.ResponseFrame{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    respBody,
	}, nil
}
