package server

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/niekvdm/digit-link/internal/db"
)

// Tunnel represents a connected client tunnel
type Tunnel struct {
	Subdomain  string
	Conn       *websocket.Conn
	CreatedAt  time.Time
	ResponseCh map[string]chan []byte // Request ID -> response channel
	mu         sync.RWMutex

	// Auth context for this tunnel
	AccountID string          // The account that owns this tunnel
	OrgID     string          // The organization this tunnel belongs to
	AppID     string          // The application ID (if persistent app)
	App       *db.Application // The application record (if persistent app)

	// Database record tracking
	RecordID string // The tunnel record ID in the database for stats tracking
}

// NewTunnel creates a new tunnel instance
func NewTunnel(subdomain string, conn *websocket.Conn) *Tunnel {
	return &Tunnel{
		Subdomain:  subdomain,
		Conn:       conn,
		CreatedAt:  time.Now(),
		ResponseCh: make(map[string]chan []byte),
	}
}

// NewTunnelWithContext creates a new tunnel with auth context
func NewTunnelWithContext(subdomain string, conn *websocket.Conn, accountID, orgID, appID string, app *db.Application) *Tunnel {
	return &Tunnel{
		Subdomain:  subdomain,
		Conn:       conn,
		CreatedAt:  time.Now(),
		ResponseCh: make(map[string]chan []byte),
		AccountID:  accountID,
		OrgID:      orgID,
		AppID:      appID,
		App:        app,
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
