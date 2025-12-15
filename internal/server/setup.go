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
	Password      string `json:"password"`
	AutoWhitelist bool   `json:"autoWhitelist"`
}

// SetupInitResponse contains the setup result with pending token for TOTP setup
type SetupInitResponse struct {
	Success      bool   `json:"success"`
	PendingToken string `json:"pendingToken,omitempty"` // For TOTP setup step
	AccountID    string `json:"accountId,omitempty"`
	Username     string `json:"username,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SetupTOTPResponse contains TOTP setup info
type SetupTOTPResponse struct {
	Success bool   `json:"success"`
	Secret  string `json:"secret,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SetupCompleteRequest contains the TOTP verification for setup completion
type SetupCompleteRequest struct {
	PendingToken string `json:"pendingToken"`
	Code         string `json:"code"`
}

// SetupCompleteResponse contains the final JWT token
type SetupCompleteResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"` // JWT for dashboard access
	Error   string `json:"error,omitempty"`
}

// handleSetup handles setup-related endpoints
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/setup/status":
		s.handleSetupStatus(w, r)
	case "/setup/init":
		s.handleSetupInit(w, r)
	case "/setup/totp":
		s.handleSetupTOTP(w, r)
	case "/setup/complete":
		s.handleSetupComplete(w, r)
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

// handleSetupInit performs initial admin setup - creates account with password
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

	// Validate username
	username := req.Username
	if username == "" {
		username = "admin"
	}

	// Validate password
	if req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Password is required",
		})
		return
	}

	if len(req.Password) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Password must be at least 8 characters",
		})
		return
	}

	// Generate a placeholder token (required by DB schema, but won't be used for login)
	_, tokenHash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	// Hash the password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Failed to create account",
		})
		return
	}

	// Create admin account with password
	account, err := s.db.CreateAccountWithPassword(username, tokenHash, passwordHash, true)
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
			normalizedIP = clientIP
		}

		_, wlErr := s.db.AddGlobalWhitelist(normalizedIP, "Auto-whitelisted during setup", account.ID)
		if wlErr != nil {
			log.Printf("Warning: Failed to auto-whitelist IP %s: %v", normalizedIP, wlErr)
		} else {
			log.Printf("Auto-whitelisted IP during setup: %s", normalizedIP)
		}
	}

	// Generate pending token for TOTP setup step
	pendingToken, err := auth.GeneratePendingToken(account.ID, username)
	if err != nil {
		log.Printf("Failed to generate pending token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupInitResponse{
			Success: false,
			Error:   "Failed to initialize TOTP setup",
		})
		return
	}

	json.NewEncoder(w).Encode(SetupInitResponse{
		Success:      true,
		PendingToken: pendingToken,
		AccountID:    account.ID,
		Username:     username,
	})
}

// handleSetupTOTP generates TOTP secret for initial admin setup
func (s *Server) handleSetupTOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get pending token from request body or query
	var pendingToken string

	// Try to parse from body first
	var body struct {
		PendingToken string `json:"pendingToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.PendingToken != "" {
		pendingToken = body.PendingToken
	} else {
		// Fall back to query parameter
		pendingToken = r.URL.Query().Get("token")
	}

	if pendingToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SetupTOTPResponse{
			Success: false,
			Error:   "Pending token required",
		})
		return
	}

	// Validate pending token
	accountID, username, err := auth.ValidatePendingToken(pendingToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SetupTOTPResponse{
			Success: false,
			Error:   "Invalid or expired token",
		})
		return
	}

	// Generate TOTP secret
	totpKey, err := auth.GenerateTOTPSecret(username)
	if err != nil {
		log.Printf("Failed to generate TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupTOTPResponse{
			Success: false,
			Error:   "Failed to generate TOTP",
		})
		return
	}

	// Encrypt and store the secret (not enabled until verified)
	encryptedSecret, err := auth.EncryptTOTPSecret(totpKey.Secret)
	if err != nil {
		log.Printf("Failed to encrypt TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupTOTPResponse{
			Success: false,
			Error:   "Failed to setup TOTP",
		})
		return
	}

	// Store the secret (not enabled until verified)
	if err := s.db.UpdateAccountTOTP(accountID, encryptedSecret, false); err != nil {
		log.Printf("Failed to store TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupTOTPResponse{
			Success: false,
			Error:   "Failed to setup TOTP",
		})
		return
	}

	json.NewEncoder(w).Encode(SetupTOTPResponse{
		Success: true,
		Secret:  totpKey.Secret,
		URL:     totpKey.URL,
	})
}

// handleSetupComplete verifies TOTP and completes setup
func (s *Server) handleSetupComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req SetupCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	if req.PendingToken == "" || req.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Token and code required",
		})
		return
	}

	// Validate pending token
	accountID, _, err := auth.ValidatePendingToken(req.PendingToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Invalid or expired token",
		})
		return
	}

	// Get account
	account, err := s.db.GetAccountByID(accountID)
	if err != nil || account == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Account not found",
		})
		return
	}

	// Decrypt the TOTP secret
	secret, err := auth.DecryptTOTPSecret(account.TOTPSecret)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Failed to verify TOTP",
		})
		return
	}

	// Validate the code
	if !auth.ValidateTOTPWithWindow(secret, req.Code) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Invalid TOTP code",
		})
		return
	}

	// Enable TOTP
	if err := s.db.UpdateAccountTOTP(accountID, account.TOTPSecret, true); err != nil {
		log.Printf("Failed to enable TOTP: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Failed to enable TOTP",
		})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(account.ID, account.Username, account.IsAdmin)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SetupCompleteResponse{
			Success: false,
			Error:   "Failed to generate session",
		})
		return
	}

	log.Printf("Setup completed for admin: %s (TOTP enabled)", account.Username)

	// Update last used
	s.db.UpdateAccountLastUsed(accountID)

	json.NewEncoder(w).Encode(SetupCompleteResponse{
		Success: true,
		Token:   token,
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
