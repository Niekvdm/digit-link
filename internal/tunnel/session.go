package tunnel

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

// DefaultYamuxConfig returns the default yamux configuration
func DefaultYamuxConfig() *yamux.Config {
	config := yamux.DefaultConfig()
	config.AcceptBacklog = 256
	config.EnableKeepAlive = true
	config.KeepAliveInterval = 30 * time.Second
	config.ConnectionWriteTimeout = 10 * time.Second
	config.StreamCloseTimeout = 5 * time.Minute
	config.StreamOpenTimeout = 30 * time.Second
	config.MaxStreamWindowSize = 256 * 1024 // 256KB
	return config
}

// Session wraps a yamux session with additional tunnel-specific state
type Session struct {
	*yamux.Session
	conn      net.Conn
	forwards  map[string]int // subdomain -> localPort
	accountID string
	orgID     string
	appID     string
	createdAt time.Time
	mu        sync.RWMutex
}

// NewServerSession creates a new server-side session from an incoming connection
func NewServerSession(conn net.Conn, config *yamux.Config) (*Session, error) {
	if config == nil {
		config = DefaultYamuxConfig()
	}

	session, err := yamux.Server(conn, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create yamux server session: %w", err)
	}

	return &Session{
		Session:   session,
		conn:      conn,
		forwards:  make(map[string]int),
		createdAt: time.Now(),
	}, nil
}

// NewClientSession creates a new client-side session to a server
func NewClientSession(conn net.Conn, config *yamux.Config) (*Session, error) {
	if config == nil {
		config = DefaultYamuxConfig()
	}

	session, err := yamux.Client(conn, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create yamux client session: %w", err)
	}

	return &Session{
		Session:   session,
		conn:      conn,
		forwards:  make(map[string]int),
		createdAt: time.Now(),
	}, nil
}

// SetForwards sets the subdomain to local port mapping
func (s *Session) SetForwards(forwards []ForwardConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forwards = make(map[string]int)
	for _, f := range forwards {
		s.forwards[f.Subdomain] = f.LocalPort
	}
}

// GetLocalPort returns the local port for a subdomain
func (s *Session) GetLocalPort(subdomain string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	port, ok := s.forwards[subdomain]
	return port, ok
}

// GetSubdomains returns all registered subdomains
func (s *Session) GetSubdomains() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subdomains := make([]string, 0, len(s.forwards))
	for subdomain := range s.forwards {
		subdomains = append(subdomains, subdomain)
	}
	return subdomains
}

// SetAccountInfo sets the account information for the session
func (s *Session) SetAccountInfo(accountID, orgID, appID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountID = accountID
	s.orgID = orgID
	s.appID = appID
}

// GetAccountInfo returns the account information
func (s *Session) GetAccountInfo() (accountID, orgID, appID string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountID, s.orgID, s.appID
}

// CreatedAt returns when the session was created
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// RemoteAddr returns the remote address of the connection
func (s *Session) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// TLSConfig returns a TLS config suitable for the tunnel server
func TLSServerConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}, nil
}

// TLSClientConfig returns a TLS config suitable for the tunnel client
func TLSClientConfig(serverName string, insecureSkipVerify bool) *tls.Config {
	return &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
}

// DialTLS connects to a server with TLS and returns the connection
func DialTLS(address string, config *tls.Config) (net.Conn, error) {
	conn, err := tls.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial TLS: %w", err)
	}
	return conn, nil
}
