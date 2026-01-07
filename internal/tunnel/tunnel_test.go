package tunnel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// generateTestCert generates a self-signed certificate for testing
func generateTestCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func TestAuthRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     AuthRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid single forward",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "myapp", LocalPort: 3000},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple forwards",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "frontend", LocalPort: 3000, Primary: true},
					{Subdomain: "api", LocalPort: 5001},
					{Subdomain: "admin", LocalPort: 8080},
				},
			},
			wantErr: false,
		},
		{
			name: "missing token",
			req: AuthRequest{
				Token: "",
				Forwards: []ForwardConfig{
					{Subdomain: "myapp", LocalPort: 3000},
				},
			},
			wantErr: true,
			errMsg:  "token is required",
		},
		{
			name: "no forwards",
			req: AuthRequest{
				Token:    "test-token",
				Forwards: []ForwardConfig{},
			},
			wantErr: true,
			errMsg:  "at least one forward is required",
		},
		{
			name: "empty subdomain",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "", LocalPort: 3000},
				},
			},
			wantErr: true,
			errMsg:  "subdomain is required",
		},
		{
			name: "invalid port zero",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "myapp", LocalPort: 0},
				},
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port too high",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "myapp", LocalPort: 70000},
				},
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "duplicate subdomain",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "myapp", LocalPort: 3000},
					{Subdomain: "myapp", LocalPort: 5000},
				},
			},
			wantErr: true,
			errMsg:  "duplicate subdomain",
		},
		{
			name: "multiple primaries",
			req: AuthRequest{
				Token: "test-token",
				Forwards: []ForwardConfig{
					{Subdomain: "frontend", LocalPort: 3000, Primary: true},
					{Subdomain: "api", LocalPort: 5001, Primary: true},
				},
			},
			wantErr: true,
			errMsg:  "only one forward can be marked as primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || s[:len(substr)] == substr)
}

func TestYamuxSessionCreation(t *testing.T) {
	cert, err := generateTestCert()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	// Start TLS listener
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Server goroutine
	serverDone := make(chan *Session, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Logf("accept error: %v", err)
			serverDone <- nil
			return
		}

		session, err := NewServerSession(conn, nil)
		if err != nil {
			t.Logf("server session error: %v", err)
			conn.Close()
			serverDone <- nil
			return
		}
		serverDone <- session
	}()

	// Client connection
	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, clientTLSConfig)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	clientSession, err := NewClientSession(conn, nil)
	if err != nil {
		t.Fatalf("failed to create client session: %v", err)
	}
	defer clientSession.Close()

	// Wait for server session
	serverSession := <-serverDone
	if serverSession == nil {
		t.Fatal("server session creation failed")
	}
	defer serverSession.Close()

	// Test session is functional
	if clientSession.IsClosed() {
		t.Error("client session should not be closed")
	}
	if serverSession.IsClosed() {
		t.Error("server session should not be closed")
	}
}

func TestMultiForwardRegistration(t *testing.T) {
	cert, err := generateTestCert()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	// Start TLS listener
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Server goroutine - handles auth
	serverReady := make(chan *Session, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverReady <- nil
			return
		}

		session, err := NewServerSession(conn, nil)
		if err != nil {
			conn.Close()
			serverReady <- nil
			return
		}

		// Accept auth stream
		stream, err := session.Accept()
		if err != nil {
			session.Close()
			serverReady <- nil
			return
		}

		// Read auth request
		authReq, err := ReadFrame[AuthRequest](stream)
		if err != nil {
			stream.Close()
			session.Close()
			serverReady <- nil
			return
		}

		// Validate
		if err := authReq.Validate(); err != nil {
			// Send failure response
			WriteFrame(stream, &AuthResponse{
				Success: false,
				Error:   err.Error(),
			})
			stream.Close()
			session.Close()
			serverReady <- nil
			return
		}

		// Set forwards on session
		session.SetForwards(authReq.Forwards)

		// Build tunnel info
		tunnels := make([]TunnelInfo, len(authReq.Forwards))
		for i, f := range authReq.Forwards {
			tunnels[i] = TunnelInfo{
				Subdomain: f.Subdomain,
				URL:       fmt.Sprintf("https://%s.test.local", f.Subdomain),
				LocalPort: f.LocalPort,
			}
		}

		// Send success response
		WriteFrame(stream, &AuthResponse{
			Success: true,
			Tunnels: tunnels,
		})
		stream.Close()

		serverReady <- session
	}()

	// Client connection
	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, clientTLSConfig)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	clientSession, err := NewClientSession(conn, nil)
	if err != nil {
		t.Fatalf("failed to create client session: %v", err)
	}
	defer clientSession.Close()

	// Open auth stream
	stream, err := clientSession.Open()
	if err != nil {
		t.Fatalf("failed to open auth stream: %v", err)
	}

	// Send auth request with multiple forwards
	authReq := AuthRequest{
		Token: "test-token",
		Forwards: []ForwardConfig{
			{Subdomain: "frontend", LocalPort: 3000, Primary: true},
			{Subdomain: "api", LocalPort: 5001},
			{Subdomain: "admin", LocalPort: 8080},
		},
	}

	if err := WriteFrame(stream, &authReq); err != nil {
		t.Fatalf("failed to send auth request: %v", err)
	}

	// Read response
	authResp, err := ReadFrame[AuthResponse](stream)
	if err != nil {
		t.Fatalf("failed to read auth response: %v", err)
	}
	stream.Close()

	if !authResp.Success {
		t.Fatalf("auth failed: %s", authResp.Error)
	}

	if len(authResp.Tunnels) != 3 {
		t.Errorf("expected 3 tunnels, got %d", len(authResp.Tunnels))
	}

	// Verify server session has correct forwards
	serverSession := <-serverReady
	if serverSession == nil {
		t.Fatal("server session not ready")
	}
	defer serverSession.Close()

	subdomains := serverSession.GetSubdomains()
	if len(subdomains) != 3 {
		t.Errorf("expected 3 subdomains, got %d", len(subdomains))
	}

	// Verify port lookups
	port, ok := serverSession.GetLocalPort("frontend")
	if !ok || port != 3000 {
		t.Errorf("expected port 3000 for frontend, got %d (ok=%v)", port, ok)
	}
	port, ok = serverSession.GetLocalPort("api")
	if !ok || port != 5001 {
		t.Errorf("expected port 5001 for api, got %d (ok=%v)", port, ok)
	}
}

func TestConcurrentRequests(t *testing.T) {
	cert, err := generateTestCert()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	// Start local HTTP server to forward to
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Subdomain", r.Header.Get("X-Subdomain"))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from %s", r.URL.Path)
	}))
	defer localServer.Close()

	// Start TLS listener
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Server goroutine
	serverSession := make(chan *Session, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverSession <- nil
			return
		}

		session, err := NewServerSession(conn, nil)
		if err != nil {
			conn.Close()
			serverSession <- nil
			return
		}
		serverSession <- session
	}()

	// Client connection
	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, clientTLSConfig)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	clientSession, err := NewClientSession(conn, nil)
	if err != nil {
		t.Fatalf("failed to create client session: %v", err)
	}
	defer clientSession.Close()

	server := <-serverSession
	if server == nil {
		t.Fatal("server session creation failed")
	}
	defer server.Close()

	// Send concurrent requests from server to client
	numRequests := 10
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)

	// Client goroutine to handle incoming requests
	go func() {
		for i := 0; i < numRequests; i++ {
			stream, err := clientSession.Accept()
			if err != nil {
				return
			}

			go func(s net.Conn) {
				defer s.Close()

				// Read request
				req, err := ReadFrame[RequestFrame](s)
				if err != nil {
					return
				}

				// Send response
				WriteFrame(s, &ResponseFrame{
					ID:      req.ID,
					Status:  200,
					Headers: map[string]string{"Content-Type": "text/plain"},
					Body:    []byte(fmt.Sprintf("Response to %s", req.Path)),
				})
			}(stream)
		}
	}()

	// Send requests concurrently
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			stream, err := server.Open()
			if err != nil {
				errors <- fmt.Errorf("failed to open stream %d: %w", idx, err)
				return
			}
			defer stream.Close()

			// Send request
			err = WriteFrame(stream, &RequestFrame{
				ID:        fmt.Sprintf("req-%d", idx),
				Subdomain: "test",
				Method:    "GET",
				Path:      fmt.Sprintf("/path/%d", idx),
				Headers:   map[string]string{"X-Request-ID": fmt.Sprintf("%d", idx)},
			})
			if err != nil {
				errors <- fmt.Errorf("failed to write request %d: %w", idx, err)
				return
			}

			// Read response
			resp, err := ReadFrame[ResponseFrame](stream)
			if err != nil {
				errors <- fmt.Errorf("failed to read response %d: %w", idx, err)
				return
			}

			if resp.Status != 200 {
				errors <- fmt.Errorf("request %d: expected status 200, got %d", idx, resp.Status)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestAuthFailure(t *testing.T) {
	cert, err := generateTestCert()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	// Start TLS listener
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Server goroutine - rejects invalid auth
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		session, err := NewServerSession(conn, nil)
		if err != nil {
			conn.Close()
			return
		}
		defer session.Close()

		// Accept auth stream
		stream, err := session.Accept()
		if err != nil {
			return
		}
		defer stream.Close()

		// Read auth request
		authReq, err := ReadFrame[AuthRequest](stream)
		if err != nil {
			return
		}

		// Validate - will fail for invalid token
		if authReq.Token != "valid-token" {
			WriteFrame(stream, &AuthResponse{
				Success: false,
				Error:   "Invalid token",
			})
			return
		}

		// Would not reach here for invalid token
		WriteFrame(stream, &AuthResponse{
			Success: true,
		})
	}()

	// Client connection
	clientTLSConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, clientTLSConfig)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	clientSession, err := NewClientSession(conn, nil)
	if err != nil {
		t.Fatalf("failed to create client session: %v", err)
	}
	defer clientSession.Close()

	// Open auth stream
	stream, err := clientSession.Open()
	if err != nil {
		t.Fatalf("failed to open auth stream: %v", err)
	}

	// Send auth request with invalid token
	authReq := AuthRequest{
		Token: "invalid-token",
		Forwards: []ForwardConfig{
			{Subdomain: "test", LocalPort: 3000},
		},
	}

	if err := WriteFrame(stream, &authReq); err != nil {
		t.Fatalf("failed to send auth request: %v", err)
	}

	// Read response
	authResp, err := ReadFrame[AuthResponse](stream)
	if err != nil {
		t.Fatalf("failed to read auth response: %v", err)
	}
	stream.Close()

	if authResp.Success {
		t.Error("expected auth to fail with invalid token")
	}
	if authResp.Error != "Invalid token" {
		t.Errorf("expected error 'Invalid token', got %q", authResp.Error)
	}
}

func TestFrameReadWrite(t *testing.T) {
	// Create a pipe for testing
	reader, writer := io.Pipe()

	// Write in goroutine
	go func() {
		defer writer.Close()

		req := &RequestFrame{
			ID:        "test-123",
			Subdomain: "myapp",
			Method:    "POST",
			Path:      "/api/data",
			Headers:   map[string]string{"Content-Type": "application/json"},
			Body:      []byte(`{"key": "value"}`),
		}

		if err := WriteFrame(writer, req); err != nil {
			t.Errorf("write error: %v", err)
		}
	}()

	// Read
	req, err := ReadFrame[RequestFrame](reader)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if req.ID != "test-123" {
		t.Errorf("expected ID 'test-123', got %q", req.ID)
	}
	if req.Subdomain != "myapp" {
		t.Errorf("expected subdomain 'myapp', got %q", req.Subdomain)
	}
	if req.Method != "POST" {
		t.Errorf("expected method 'POST', got %q", req.Method)
	}
	if req.Path != "/api/data" {
		t.Errorf("expected path '/api/data', got %q", req.Path)
	}
	if string(req.Body) != `{"key": "value"}` {
		t.Errorf("unexpected body: %s", req.Body)
	}
}
