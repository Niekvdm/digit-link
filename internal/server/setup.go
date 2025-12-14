package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/niekvdm/digit-link/internal/auth"
)

// SetupStatusResponse contains the setup status
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needsSetup"`
}

// SetupInitRequest contains the initial setup request
type SetupInitRequest struct {
	Username      string `json:"username"`
	AutoWhitelist bool   `json:"autoWhitelist"`
}

// SetupInitResponse contains the setup result
type SetupInitResponse struct {
	Success  bool   `json:"success"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Error    string `json:"error,omitempty"`
}

// handleSetup handles setup-related endpoints
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/setup/status":
		s.handleSetupStatus(w, r)
	case "/setup/init":
		s.handleSetupInit(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleSetupStatus checks if initial setup is needed
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	needsSetup := true

	if s.db != nil {
		hasAdmin, err := s.db.HasAdminAccount()
		if err != nil {
			log.Printf("Error checking admin status: %v", err)
			http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
			return
		}
		needsSetup = !hasAdmin
	}

	json.NewEncoder(w).Encode(SetupStatusResponse{
		NeedsSetup: needsSetup,
	})
}

// handleSetupInit performs initial admin setup
func (s *Server) handleSetupInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if s.db == nil {
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Database not configured",
		})
		return
	}

	// Check if already configured
	hasAdmin, err := s.db.HasAdminAccount()
	if err != nil {
		log.Printf("Error checking admin status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Database error",
		})
		return
	}

	if hasAdmin {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Server already configured",
		})
		return
	}

	// Parse request
	var req SetupInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Default username
	username := req.Username
	if username == "" {
		username = "admin"
	}

	// Generate token
	token, tokenHash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	// Create admin account
	account, err := s.db.CreateAccount(username, tokenHash, true)
	if err != nil {
		log.Printf("Failed to create admin account: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Failed to create account",
		})
		return
	}

	log.Printf("Initial admin account created: %s", username)

	// Auto-whitelist the client's IP if requested
	if req.AutoWhitelist {
		clientIP := auth.GetClientIP(r)
		normalizedIP, normErr := auth.NormalizeIP(clientIP)
		if normErr != nil {
			log.Printf("Warning: Failed to normalize IP %s: %v", clientIP, normErr)
			// Use the raw IP as fallback
			normalizedIP = clientIP
		}

		_, wlErr := s.db.AddGlobalWhitelist(normalizedIP, "Auto-whitelisted during setup", account.ID)
		if wlErr != nil {
			log.Printf("Warning: Failed to auto-whitelist IP %s: %v", normalizedIP, wlErr)
			// Don't fail the setup, just log the warning
		} else {
			log.Printf("Auto-whitelisted IP during setup: %s", normalizedIP)
		}
	}

	json.NewEncoder(w).Encode(SetupInitResponse{
		Success:  true,
		Token:    token,
		Username: username,
	})
}

// NeedsSetup returns true if the server needs initial setup
func (s *Server) NeedsSetup() bool {
	if s.db == nil {
		return false
	}

	hasAdmin, err := s.db.HasAdminAccount()
	if err != nil {
		log.Printf("Error checking admin status: %v", err)
		return false
	}

	return !hasAdmin
}
