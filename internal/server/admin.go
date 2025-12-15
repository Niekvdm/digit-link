package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/niekvdm/digit-link/internal/auth"
)

// handleAdmin routes admin API requests
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	// Verify admin authentication
	account, err := s.authenticateAdmin(r)
	if err != nil || account == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Route admin endpoints
	path := strings.TrimPrefix(r.URL.Path, "/admin")

	switch {
	// Account management
	case path == "/accounts" && r.Method == http.MethodGet:
		s.handleListAccounts(w, r)
	case path == "/accounts" && r.Method == http.MethodPost:
		s.handleCreateAccount(w, r)
	case strings.HasPrefix(path, "/accounts/") && r.Method == http.MethodDelete:
		accountID := strings.TrimPrefix(path, "/accounts/")
		s.handleDeleteAccount(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/regenerate") && r.Method == http.MethodPost:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/regenerate")
		s.handleRegenerateToken(w, r, accountID)

	// Whitelist management
	case path == "/whitelist" && r.Method == http.MethodGet:
		s.handleListWhitelist(w, r)
	case path == "/whitelist" && r.Method == http.MethodPost:
		s.handleAddWhitelist(w, r, account.ID)
	case strings.HasPrefix(path, "/whitelist/") && r.Method == http.MethodDelete:
		entryID := strings.TrimPrefix(path, "/whitelist/")
		s.handleDeleteWhitelist(w, r, entryID)

	// Tunnel management
	case path == "/tunnels" && r.Method == http.MethodGet:
		s.handleListTunnels(w, r)

	// Stats
	case path == "/stats" && r.Method == http.MethodGet:
		s.handleStats(w, r)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// authenticateAdmin verifies the admin authentication from the request
// Supports both JWT tokens (for dashboard) and API tokens (for clients)
func (s *Server) authenticateAdmin(r *http.Request) (*struct {
	ID       string
	Username string
	IsAdmin  bool
}, error) {
	if s.db == nil {
		return nil, nil
	}

	// Get token from header
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		return nil, nil
	}

	// First, try to validate as JWT token
	claims, err := auth.ValidateJWT(token)
	if err == nil && claims != nil {
		// Valid JWT token
		if !claims.IsAdmin {
			return nil, nil
		}
		return &struct {
			ID       string
			Username string
			IsAdmin  bool
		}{
			ID:       claims.AccountID,
			Username: claims.Username,
			IsAdmin:  claims.IsAdmin,
		}, nil
	}

	// Fall back to API token validation (for backward compatibility)
	tokenHash := auth.HashToken(token)
	account, err := s.db.GetAccountByTokenHash(tokenHash)
	if err != nil {
		return nil, err
	}
	if account == nil || !account.IsAdmin {
		return nil, nil
	}

	return &struct {
		ID       string
		Username string
		IsAdmin  bool
	}{
		ID:       account.ID,
		Username: account.Username,
		IsAdmin:  account.IsAdmin,
	}, nil
}

// handleListAccounts returns all accounts
func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.db.ListAccounts()
	if err != nil {
		log.Printf("Failed to list accounts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Don't expose token hashes
	result := make([]map[string]interface{}, len(accounts))
	for i, acc := range accounts {
		result[i] = map[string]interface{}{
			"id":          acc.ID,
			"username":    acc.Username,
			"isAdmin":     acc.IsAdmin,
			"totpEnabled": acc.TOTPEnabled,
			"createdAt":   acc.CreatedAt,
			"lastUsed":    acc.LastUsed,
			"active":      acc.Active,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": result,
	})
}

// handleCreateAccount creates a new account
func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		IsAdmin  bool   `json:"isAdmin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	existing, err := s.db.GetAccountByUsername(req.Username)
	if err != nil {
		log.Printf("Failed to check username: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Generate token
	token, tokenHash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create account
	account, err := s.db.CreateAccount(req.Username, tokenHash, req.IsAdmin)
	if err != nil {
		log.Printf("Failed to create account: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Account created: %s (admin: %v)", req.Username, req.IsAdmin)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"account": map[string]interface{}{
			"id":        account.ID,
			"username":  account.Username,
			"isAdmin":   account.IsAdmin,
			"createdAt": account.CreatedAt,
		},
		"token": token, // Only returned once at creation
	})
}

// handleDeleteAccount deactivates an account
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if err := s.db.DeactivateAccount(accountID); err != nil {
		log.Printf("Failed to deactivate account: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Account deactivated: %s", accountID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleRegenerateToken generates a new token for an account
func (s *Server) handleRegenerateToken(w http.ResponseWriter, r *http.Request, accountID string) {
	// Generate new token
	token, tokenHash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update account
	if err := s.db.UpdateAccountToken(accountID, tokenHash); err != nil {
		log.Printf("Failed to update token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Token regenerated for account: %s", accountID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// handleListWhitelist returns all global whitelist entries
func (s *Server) handleListWhitelist(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.ListGlobalWhitelist()
	if err != nil {
		log.Printf("Failed to list whitelist: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
	})
}

// handleAddWhitelist adds an IP range to the global whitelist
func (s *Server) handleAddWhitelist(w http.ResponseWriter, r *http.Request, createdBy string) {
	var req struct {
		IPRange     string `json:"ipRange"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.IPRange == "" {
		http.Error(w, "IP range is required", http.StatusBadRequest)
		return
	}

	entry, err := s.db.AddGlobalWhitelist(req.IPRange, req.Description, createdBy)
	if err != nil {
		log.Printf("Failed to add whitelist entry: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Whitelist entry added: %s (%s)", req.IPRange, req.Description)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entry":   entry,
	})
}

// handleDeleteWhitelist removes an IP range from the global whitelist
func (s *Server) handleDeleteWhitelist(w http.ResponseWriter, r *http.Request, entryID string) {
	if err := s.db.DeleteGlobalWhitelist(entryID); err != nil {
		log.Printf("Failed to delete whitelist entry: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Whitelist entry deleted: %s", entryID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleListTunnels returns active tunnels
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	// Get in-memory active tunnels
	activeTunnels := s.GetActiveTunnels()

	// Get database tunnel records for additional info
	var dbTunnels interface{}
	if s.db != nil {
		if tunnels, err := s.db.ListActiveTunnels(); err == nil {
			dbTunnels = tunnels
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active":  activeTunnels,
		"records": dbTunnels,
	})
}

// handleStats returns server statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tunnelCount := len(s.tunnels)
	s.mu.RUnlock()

	stats := map[string]interface{}{
		"activeTunnels": tunnelCount,
	}

	if s.db != nil {
		if count, err := s.db.CountAccounts(); err == nil {
			stats["totalAccounts"] = count
		}
		if count, err := s.db.CountActiveAccounts(); err == nil {
			stats["activeAccounts"] = count
		}
		if count, err := s.db.CountGlobalWhitelist(); err == nil {
			stats["whitelistEntries"] = count
		}
		if total, sent, received, err := s.db.GetTotalTunnelStats(); err == nil {
			stats["totalTunnels"] = total
			stats["totalBytesSent"] = sent
			stats["totalBytesReceived"] = received
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
