package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/niekvdm/digit-link/internal/auth"
)

// maxAuthRequestBodySize is the maximum allowed request body size for auth endpoints (64KB)
const maxAuthRequestBodySize = 64 * 1024

// validateAuthJSONRequest validates Content-Type and limits request body size
// Returns true if valid, false otherwise (and sends error response)
func validateAuthJSONRequest(w http.ResponseWriter, r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" || (!strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "text/json")) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Content-Type must be application/json"})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBodySize)
	return true
}

// LoginRequest contains the login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse contains the login result
type LoginResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`        // Final JWT token (when no TOTP required)
	PendingToken string `json:"pendingToken,omitempty"` // Pending token for TOTP step
	NeedsTOTP    bool   `json:"needsTotp,omitempty"`
	NeedsSetup   bool   `json:"needsSetup,omitempty"`
	AccountType  string `json:"accountType,omitempty"` // "admin" or "org"
	OrgID        string `json:"orgId,omitempty"`       // For org accounts
	OrgName      string `json:"orgName,omitempty"`     // Organization name
	IsOrgAdmin   bool   `json:"isOrgAdmin,omitempty"`  // Is org admin
	Error        string `json:"error,omitempty"`
}

// TOTPSetupRequest contains the TOTP setup verification
type TOTPSetupRequest struct {
	PendingToken string `json:"pendingToken"`
	Code         string `json:"code"`
}

// TOTPSetupResponse contains the TOTP setup result
type TOTPSetupResponse struct {
	Success     bool   `json:"success"`
	Secret      string `json:"secret,omitempty"`
	URL         string `json:"url,omitempty"`
	Token       string `json:"token,omitempty"`
	AccountType string `json:"accountType,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	OrgName     string `json:"orgName,omitempty"`
	IsOrgAdmin  bool   `json:"isOrgAdmin,omitempty"`
	Error       string `json:"error,omitempty"`
}

// TOTPVerifyRequest contains the TOTP verification
type TOTPVerifyRequest struct {
	PendingToken string `json:"pendingToken"`
	Code         string `json:"code"`
}

// TOTPVerifyResponse contains the TOTP verification result
type TOTPVerifyResponse struct {
	Success     bool   `json:"success"`
	Token       string `json:"token,omitempty"`
	AccountType string `json:"accountType,omitempty"`
	OrgID       string `json:"orgId,omitempty"`
	OrgName     string `json:"orgName,omitempty"`
	IsOrgAdmin  bool   `json:"isOrgAdmin,omitempty"`
	Error       string `json:"error,omitempty"`
}

// CheckAccountRequest contains the username to check
type CheckAccountRequest struct {
	Username string `json:"username"`
}

// CheckAccountResponse contains account metadata for login flow
type CheckAccountResponse struct {
	Exists       bool   `json:"exists"`
	AccountType  string `json:"accountType,omitempty"`  // "admin" or "org"
	RequiresTOTP bool   `json:"requiresTotp,omitempty"` // Based on account + org policy
	OrgName      string `json:"orgName,omitempty"`      // For org accounts
}

// handleAuth routes authentication endpoints
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/auth")

	switch {
	case path == "/check-account" && r.Method == http.MethodPost:
		s.handleCheckAccount(w, r)
	case path == "/login" && r.Method == http.MethodPost:
		s.handleLogin(w, r)
	case path == "/org/login" && r.Method == http.MethodPost:
		s.handleOrgLogin(w, r)
	case path == "/totp/setup" && r.Method == http.MethodGet:
		s.handleTOTPSetupGet(w, r)
	case path == "/totp/setup" && r.Method == http.MethodPost:
		s.handleTOTPSetupPost(w, r)
	case path == "/totp/verify" && r.Method == http.MethodPost:
		s.handleTOTPVerify(w, r)
	default:
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
	}
}

// handleCheckAccount validates username and returns account metadata for login flow
func (s *Server) handleCheckAccount(w http.ResponseWriter, r *http.Request) {
	// Apply rate limiting to prevent username enumeration attacks
	if s.loginRateLimiter != nil {
		clientIP := auth.GetClientIP(r)
		key := auth.IPRateLimitKey(clientIP)
		allowed, retryAfter := s.loginRateLimiter.Allow(key)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests. Please try again later."})
			return
		}
	}

	var req CheckAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CheckAccountResponse{Exists: false})
		return
	}

	if req.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CheckAccountResponse{Exists: false})
		return
	}

	if s.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CheckAccountResponse{Exists: false})
		return
	}

	// Get account by username
	account, err := s.db.GetAccountByUsername(req.Username)
	if err != nil {
		log.Printf("Check account error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CheckAccountResponse{Exists: false})
		return
	}

	if account == nil || !account.Active {
		// Don't reveal if account exists but is inactive
		json.NewEncoder(w).Encode(CheckAccountResponse{Exists: false})
		return
	}

	// Check if account has password set (required for login flow)
	if account.PasswordHash == "" {
		json.NewEncoder(w).Encode(CheckAccountResponse{Exists: false})
		return
	}

	// Determine account type and TOTP requirements
	resp := CheckAccountResponse{
		Exists: true,
	}

	if account.IsAdmin {
		resp.AccountType = "admin"
		// Admins always require TOTP (either setup or verify)
		resp.RequiresTOTP = true
	} else if account.OrgID != "" {
		resp.AccountType = "org"
		// Check if TOTP is enabled on the account
		resp.RequiresTOTP = account.TOTPEnabled

		// Check org-level TOTP requirement
		if org, err := s.db.GetOrganizationByID(account.OrgID); err == nil && org != nil {
			resp.OrgName = org.Name
			if org.RequireTOTP {
				resp.RequiresTOTP = true
			}
		}
	} else {
		// Account without org - treat as regular user
		resp.AccountType = "user"
		resp.RequiresTOTP = account.TOTPEnabled
	}

	json.NewEncoder(w).Encode(resp)
}

// handleLogin handles username/password authentication for both admin and org accounts
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !validateAuthJSONRequest(w, r) {
		return
	}

	// Apply rate limiting
	if s.loginRateLimiter != nil {
		clientIP := auth.GetClientIP(r)
		key := auth.IPRateLimitKey(clientIP)
		allowed, retryAfter := s.loginRateLimiter.Allow(key)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(LoginResponse{Error: "Too many login attempts. Please try again later."})
			return
		}
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Invalid request"})
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Username and password required"})
		return
	}

	if s.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Database not configured"})
		return
	}

	// Get account by username
	account, err := s.db.GetAccountByUsername(req.Username)
	if err != nil {
		log.Printf("Login error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Internal error"})
		return
	}

	if account == nil || !account.Active {
		// Record failed attempt for rate limiting
		if s.loginRateLimiter != nil {
			clientIP := auth.GetClientIP(r)
			s.loginRateLimiter.RecordFailure(auth.IPRateLimitKey(clientIP))
		}
		// Use same error message to prevent username enumeration
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Invalid credentials"})
		return
	}

	// Check if account has password set
	if account.PasswordHash == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Account not configured for password login"})
		return
	}

	// Verify password
	if !auth.VerifyPassword(req.Password, account.PasswordHash) {
		// Record failed attempt for rate limiting
		if s.loginRateLimiter != nil {
			clientIP := auth.GetClientIP(r)
			s.loginRateLimiter.RecordFailure(auth.IPRateLimitKey(clientIP))
		}
		log.Printf("Failed login attempt from IP: %s", auth.GetClientIP(r))
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Invalid credentials"})
		return
	}

	// Record successful login for rate limiting
	if s.loginRateLimiter != nil {
		clientIP := auth.GetClientIP(r)
		s.loginRateLimiter.RecordSuccess(auth.IPRateLimitKey(clientIP))
	}

	// Determine account type
	accountType := "user"
	if account.IsAdmin {
		accountType = "admin"
	} else if account.OrgID != "" {
		accountType = "org"
	}

	// Determine if TOTP is required
	requiresTOTP := account.TOTPEnabled
	if account.IsAdmin {
		// Admins always require TOTP
		requiresTOTP = true
	} else if account.OrgID != "" {
		// Check org-level TOTP requirement
		if org, err := s.db.GetOrganizationByID(account.OrgID); err == nil && org != nil && org.RequireTOTP {
			requiresTOTP = true
		}
	}

	// If TOTP not required and not enabled, issue token directly
	if !requiresTOTP && !account.TOTPEnabled {
		// Generate JWT token directly (no TOTP step)
		token, err := auth.GenerateJWTWithOrg(account.ID, account.Username, account.IsAdmin, account.OrgID)
		if err != nil {
			log.Printf("Failed to generate JWT: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(LoginResponse{Error: "Internal error"})
			return
		}

		// Get org name if org user
		var orgName string
		if account.OrgID != "" {
			if org, _ := s.db.GetOrganizationByID(account.OrgID); org != nil {
				orgName = org.Name
			}
		}

		log.Printf("Successful login for user: %s (type: %s, no TOTP)", account.Username, accountType)
		s.db.UpdateAccountLastUsed(account.ID)

		json.NewEncoder(w).Encode(LoginResponse{
			Success:     true,
			Token:       token,
			AccountType: accountType,
			OrgID:       account.OrgID,
			OrgName:     orgName,
			IsOrgAdmin:  account.IsOrgAdmin,
		})
		return
	}

	// TOTP is required - generate pending token for TOTP step
	pendingToken, err := auth.GeneratePendingToken(account.ID, account.Username)
	if err != nil {
		log.Printf("Failed to generate pending token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{Error: "Internal error"})
		return
	}

	// Check if TOTP is set up
	if !account.TOTPEnabled || account.TOTPSecret == "" {
		// User needs to set up TOTP
		json.NewEncoder(w).Encode(LoginResponse{
			Success:      true,
			PendingToken: pendingToken,
			NeedsSetup:   true,
			AccountType:  accountType,
			OrgID:        account.OrgID,
		})
		return
	}

	// User has TOTP enabled, needs to verify
	json.NewEncoder(w).Encode(LoginResponse{
		Success:      true,
		PendingToken: pendingToken,
		NeedsTOTP:    true,
		AccountType:  accountType,
		OrgID:        account.OrgID,
	})
}

// handleTOTPSetupGet generates a new TOTP secret for setup
func (s *Server) handleTOTPSetupGet(w http.ResponseWriter, r *http.Request) {
	pendingToken := r.URL.Query().Get("token")
	if pendingToken == "" {
		// Try header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			pendingToken = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if pendingToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Pending token required"})
		return
	}

	// Validate pending token
	accountID, username, err := auth.ValidatePendingToken(pendingToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Invalid or expired token"})
		return
	}

	// Generate TOTP secret
	totpKey, err := auth.GenerateTOTPSecret(username)
	if err != nil {
		log.Printf("Failed to generate TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Failed to generate TOTP"})
		return
	}

	// Encrypt and store the secret (but don't enable yet)
	encryptedSecret, err := auth.EncryptTOTPSecret(totpKey.Secret)
	if err != nil {
		log.Printf("Failed to encrypt TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Failed to setup TOTP"})
		return
	}

	// Store the secret (not enabled until verified)
	if err := s.db.UpdateAccountTOTP(accountID, encryptedSecret, false); err != nil {
		log.Printf("Failed to store TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Failed to setup TOTP"})
		return
	}

	json.NewEncoder(w).Encode(TOTPSetupResponse{
		Success: true,
		Secret:  totpKey.Secret,
		URL:     totpKey.URL,
	})
}

// handleTOTPSetupPost verifies the TOTP code and enables TOTP
func (s *Server) handleTOTPSetupPost(w http.ResponseWriter, r *http.Request) {
	var req TOTPSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Invalid request"})
		return
	}

	if req.PendingToken == "" || req.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Token and code required"})
		return
	}

	// Validate pending token
	accountID, _, err := auth.ValidatePendingToken(req.PendingToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Invalid or expired token"})
		return
	}

	// Get account
	account, err := s.db.GetAccountByID(accountID)
	if err != nil || account == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Account not found"})
		return
	}

	// Decrypt the TOTP secret
	secret, err := auth.DecryptTOTPSecret(account.TOTPSecret)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Failed to verify TOTP"})
		return
	}

	// Validate the code
	if !auth.ValidateTOTPWithWindow(secret, req.Code) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Invalid TOTP code"})
		return
	}

	// Enable TOTP
	if err := s.db.UpdateAccountTOTP(accountID, account.TOTPSecret, true); err != nil {
		log.Printf("Failed to enable TOTP: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Failed to enable TOTP"})
		return
	}

	// Generate JWT token with org context
	token, err := auth.GenerateJWTWithOrg(account.ID, account.Username, account.IsAdmin, account.OrgID)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPSetupResponse{Error: "Failed to generate session"})
		return
	}

	// Determine account type
	accountType := "user"
	if account.IsAdmin {
		accountType = "admin"
	} else if account.OrgID != "" {
		accountType = "org"
	}

	// Get org name if org user
	var orgName string
	if account.OrgID != "" {
		if org, _ := s.db.GetOrganizationByID(account.OrgID); org != nil {
			orgName = org.Name
		}
	}

	log.Printf("TOTP enabled for user: %s (type: %s)", account.Username, accountType)

	// Update last used
	s.db.UpdateAccountLastUsed(accountID)

	json.NewEncoder(w).Encode(TOTPSetupResponse{
		Success:     true,
		Token:       token,
		AccountType: accountType,
		OrgID:       account.OrgID,
		OrgName:     orgName,
		IsOrgAdmin:  account.IsOrgAdmin,
	})
}

// handleTOTPVerify verifies the TOTP code and issues JWT
func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	if !validateAuthJSONRequest(w, r) {
		return
	}

	// Apply rate limiting (use account-specific key if possible, fall back to IP)
	var rateLimitKey string
	if s.loginRateLimiter != nil {
		clientIP := auth.GetClientIP(r)
		rateLimitKey = auth.IPRateLimitKey(clientIP)
		allowed, retryAfter := s.loginRateLimiter.Allow(rateLimitKey)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Too many verification attempts. Please try again later."})
			return
		}
	}

	var req TOTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Invalid request"})
		return
	}

	if req.PendingToken == "" || req.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Token and code required"})
		return
	}

	// Validate pending token
	accountID, _, err := auth.ValidatePendingToken(req.PendingToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Invalid or expired token"})
		return
	}

	// Get account
	account, err := s.db.GetAccountByID(accountID)
	if err != nil || account == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Account not found"})
		return
	}

	if !account.TOTPEnabled || account.TOTPSecret == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "TOTP not configured"})
		return
	}

	// Decrypt the TOTP secret
	secret, err := auth.DecryptTOTPSecret(account.TOTPSecret)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Failed to verify TOTP"})
		return
	}

	// Validate the code
	if !auth.ValidateTOTPWithWindow(secret, req.Code) {
		// Record failed attempt for rate limiting
		if s.loginRateLimiter != nil && rateLimitKey != "" {
			s.loginRateLimiter.RecordFailure(rateLimitKey)
		}
		log.Printf("Invalid TOTP code from IP: %s", auth.GetClientIP(r))
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Invalid TOTP code"})
		return
	}

	// Record successful verification for rate limiting
	if s.loginRateLimiter != nil && rateLimitKey != "" {
		s.loginRateLimiter.RecordSuccess(rateLimitKey)
	}

	// Generate JWT token with org context
	token, err := auth.GenerateJWTWithOrg(account.ID, account.Username, account.IsAdmin, account.OrgID)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(TOTPVerifyResponse{Error: "Failed to generate session"})
		return
	}

	// Determine account type
	accountType := "user"
	if account.IsAdmin {
		accountType = "admin"
	} else if account.OrgID != "" {
		accountType = "org"
	}

	// Get org name if org user
	var orgName string
	if account.OrgID != "" {
		if org, _ := s.db.GetOrganizationByID(account.OrgID); org != nil {
			orgName = org.Name
		}
	}

	log.Printf("Successful login for user: %s (type: %s)", account.Username, accountType)

	// Update last used
	s.db.UpdateAccountLastUsed(accountID)

	json.NewEncoder(w).Encode(TOTPVerifyResponse{
		Success:     true,
		Token:       token,
		AccountType: accountType,
		OrgID:       account.OrgID,
		OrgName:     orgName,
		IsOrgAdmin:  account.IsOrgAdmin,
	})
}

// OrgLoginRequest contains the org account login credentials
type OrgLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// OrgLoginResponse contains the org login result
type OrgLoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	OrgID   string `json:"orgId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleOrgLogin handles organization account username/password authentication
// Org accounts don't require TOTP - they use simpler password-only authentication
func (s *Server) handleOrgLogin(w http.ResponseWriter, r *http.Request) {
	if !validateAuthJSONRequest(w, r) {
		return
	}

	// Apply rate limiting
	if s.loginRateLimiter != nil {
		clientIP := auth.GetClientIP(r)
		key := auth.IPRateLimitKey(clientIP)
		allowed, retryAfter := s.loginRateLimiter.Allow(key)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Too many login attempts. Please try again later."})
			return
		}
	}

	var req OrgLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Invalid request"})
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Username and password required"})
		return
	}

	if s.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Database not configured"})
		return
	}

	// Get account by username
	account, err := s.db.GetAccountByUsername(req.Username)
	if err != nil {
		log.Printf("Org login error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Internal error"})
		return
	}

	// Org accounts must NOT be admin and MUST have org_id
	if account == nil || !account.Active || account.IsAdmin || account.OrgID == "" {
		// Record failed attempt for rate limiting
		if s.loginRateLimiter != nil {
			clientIP := auth.GetClientIP(r)
			s.loginRateLimiter.RecordFailure(auth.IPRateLimitKey(clientIP))
		}
		// Use same error message to prevent username enumeration
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Invalid credentials"})
		return
	}

	// Check if account has password set
	if account.PasswordHash == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Account not configured for password login"})
		return
	}

	// Verify password
	if !auth.VerifyPassword(req.Password, account.PasswordHash) {
		// Record failed attempt for rate limiting
		if s.loginRateLimiter != nil {
			clientIP := auth.GetClientIP(r)
			s.loginRateLimiter.RecordFailure(auth.IPRateLimitKey(clientIP))
		}
		log.Printf("Failed org login attempt from IP: %s", auth.GetClientIP(r))
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Invalid credentials"})
		return
	}

	// Generate JWT token with org context (no TOTP required for org accounts)
	token, err := auth.GenerateJWTWithOrg(account.ID, account.Username, false, account.OrgID)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(OrgLoginResponse{Error: "Internal error"})
		return
	}

	// Record successful login for rate limiting
	if s.loginRateLimiter != nil {
		clientIP := auth.GetClientIP(r)
		s.loginRateLimiter.RecordSuccess(auth.IPRateLimitKey(clientIP))
	}

	log.Printf("Successful org login for user: %s (org: %s)", account.Username, account.OrgID)

	// Update last used
	s.db.UpdateAccountLastUsed(account.ID)

	json.NewEncoder(w).Encode(OrgLoginResponse{
		Success: true,
		Token:   token,
		OrgID:   account.OrgID,
	})
}
