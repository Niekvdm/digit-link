package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/niekvdm/digit-link/internal/protocol"
)

// Proxy handles forwarding requests to the local service
type Proxy struct {
	localAddr string
	client    *http.Client
}

// DefaultTimeout is the default timeout for forwarding requests (5 minutes)
const DefaultTimeout = 5 * time.Minute

// NewProxy creates a new local proxy
func NewProxy(localPort int) *Proxy {
	return NewProxyWithTimeout(localPort, DefaultTimeout)
}

// NewProxyWithTimeout creates a new local proxy with a custom timeout
func NewProxyWithTimeout(localPort int, timeout time.Duration) *Proxy {
	return &Proxy{
		localAddr: fmt.Sprintf("http://localhost:%d", localPort),
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
