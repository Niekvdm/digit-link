package server

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Tunnel represents a connected client tunnel
type Tunnel struct {
	Subdomain  string
	Conn       *websocket.Conn
	ResponseCh map[string]chan []byte // Request ID -> response channel
	mu         sync.RWMutex
}

// NewTunnel creates a new tunnel instance
func NewTunnel(subdomain string, conn *websocket.Conn) *Tunnel {
	return &Tunnel{
		Subdomain:  subdomain,
		Conn:       conn,
		ResponseCh: make(map[string]chan []byte),
	}
}

// AddResponseChannel creates a channel for a request ID
func (t *Tunnel) AddResponseChannel(requestID string) chan []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	ch := make(chan []byte, 1)
	t.ResponseCh[requestID] = ch
	return ch
}

// GetResponseChannel retrieves and removes a response channel
func (t *Tunnel) GetResponseChannel(requestID string) (chan []byte, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ch, ok := t.ResponseCh[requestID]
	if ok {
		delete(t.ResponseCh, requestID)
	}
	return ch, ok
}

// RemoveResponseChannel removes a response channel (for cleanup)
func (t *Tunnel) RemoveResponseChannel(requestID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if ch, ok := t.ResponseCh[requestID]; ok {
		close(ch)
		delete(t.ResponseCh, requestID)
	}
}

// Close closes the tunnel and all pending response channels
func (t *Tunnel) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, ch := range t.ResponseCh {
		close(ch)
		delete(t.ResponseCh, id)
	}
	t.Conn.Close()
}
