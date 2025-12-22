package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// HealthResponse represents the response from the /health endpoint
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// ReadyResponse represents the response from the /ready endpoint
type ReadyResponse struct {
	Status string `json:"status"`
}

// LiveResponse represents the response from the /live endpoint
type LiveResponse struct {
	Status string `json:"status"`
}

// databaseHealthCheckTimeout is the timeout for database health checks
const databaseHealthCheckTimeout = 2 * time.Second

// checkDatabaseHealth verifies the database connection is healthy
func (s *Server) checkDatabaseHealth(ctx context.Context) error {
	if s.db == nil {
		return nil // No database configured, consider healthy
	}

	ctx, cancel := context.WithTimeout(ctx, databaseHealthCheckTimeout)
	defer cancel()

	return s.db.Conn().PingContext(ctx)
}

// handleHealthCheck handles the /health endpoint for the health check server
// Returns overall system health status with database connectivity check
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	checks := make(map[string]string)
	healthy := true

	// Check database connectivity
	if err := s.checkDatabaseHealth(r.Context()); err != nil {
		checks["database"] = "disconnected"
		healthy = false
		log.Printf("Health check: database unhealthy: %v", err)
	} else {
		checks["database"] = "connected"
	}

	response := HealthResponse{
		Checks: checks,
	}

	if healthy {
		response.Status = "ok"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

// handleReadyCheck handles the /ready endpoint for the health check server
// Returns whether the server is ready to accept traffic (Kubernetes readiness probe)
func (s *Server) handleReadyCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ready := true

	// Check database connectivity - server not ready if DB is unavailable
	if err := s.checkDatabaseHealth(r.Context()); err != nil {
		ready = false
		log.Printf("Readiness check: database unavailable: %v", err)
	}

	response := ReadyResponse{}

	if ready {
		response.Status = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode ready response: %v", err)
	}
}

// handleLiveCheck handles the /live endpoint for the health check server
// Returns whether the process is alive (Kubernetes liveness probe)
// This endpoint should not check external dependencies
func (s *Server) handleLiveCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := LiveResponse{
		Status: "alive",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode live response: %v", err)
	}
}

// GetHealthCheckPort returns the health check server port from environment or default (8081)
func GetHealthCheckPort() string {
	if port := os.Getenv("HEALTH_CHECK_PORT"); port != "" {
		return port
	}
	return "8081"
}

// StartHealthCheckServer starts a separate HTTP server for health check endpoints
// Returns the server instance for graceful shutdown capability
func (s *Server) StartHealthCheckServer() *http.Server {
	port := GetHealthCheckPort()
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealthCheck)
	mux.HandleFunc("/ready", s.handleReadyCheck)
	mux.HandleFunc("/live", s.handleLiveCheck)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Health check server listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health check server error: %v", err)
		}
	}()

	return server
}
