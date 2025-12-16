package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
)

// maxRequestBodySize is the maximum allowed request body size (1MB)
const maxRequestBodySize = 1 << 20

// jsonError writes a JSON error response
func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// jsonResponse writes a JSON response with error logging
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// validateJSONContentType checks if the request has a valid JSON content type
// Returns true if valid, false otherwise (and sends error response)
func validateJSONContentType(w http.ResponseWriter, r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" || (!strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "text/json")) {
		jsonError(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

// limitRequestBody wraps the request body with a size limit
func limitRequestBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodySize)
}

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
	// Self-management endpoints (admin's own account)
	case path == "/me" && r.Method == http.MethodGet:
		s.handleAdminGetMe(w, r, account)
	case path == "/me/password" && r.Method == http.MethodPut:
		s.handleAdminSetMyPassword(w, r, account)
	case path == "/me/totp/setup" && r.Method == http.MethodGet:
		s.handleAdminGetMyTOTPSetup(w, r, account)
	case path == "/me/totp/setup" && r.Method == http.MethodPost:
		s.handleAdminEnableMyTOTP(w, r, account)
	case path == "/me/totp" && r.Method == http.MethodDelete:
		s.handleAdminDisableMyTOTP(w, r, account)

	// Account management
	case path == "/accounts" && r.Method == http.MethodGet:
		s.handleListAccounts(w, r)
	case path == "/accounts" && r.Method == http.MethodPost:
		s.handleCreateAccount(w, r)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/hard") && r.Method == http.MethodDelete:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/hard")
		s.handleHardDeleteAccount(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/activate") && r.Method == http.MethodPost:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/activate")
		s.handleActivateAccount(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/regenerate") && r.Method == http.MethodPost:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/regenerate")
		s.handleRegenerateToken(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/organization") && r.Method == http.MethodPut:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/organization")
		s.handleSetAccountOrganization(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/password") && r.Method == http.MethodPut:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/password")
		s.handleSetAccountPassword(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/username") && r.Method == http.MethodPut:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/username")
		s.handleSetAccountUsername(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/org-admin") && r.Method == http.MethodPut:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/org-admin")
		s.handleSetAccountOrgAdmin(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/totp") && r.Method == http.MethodDelete:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/totp")
		s.handleResetAccountTOTP(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && r.Method == http.MethodGet:
		accountID := strings.TrimPrefix(path, "/accounts/")
		s.handleGetAccount(w, r, accountID)
	case strings.HasPrefix(path, "/accounts/") && r.Method == http.MethodDelete:
		accountID := strings.TrimPrefix(path, "/accounts/")
		s.handleDeleteAccount(w, r, accountID)

	// Whitelist management (global - legacy, kept for backward compatibility)
	case path == "/whitelist" && r.Method == http.MethodGet:
		s.handleListWhitelist(w, r)
	case path == "/whitelist" && r.Method == http.MethodPost:
		s.handleAddWhitelist(w, r, account.ID)
	case strings.HasPrefix(path, "/whitelist/") && r.Method == http.MethodDelete:
		entryID := strings.TrimPrefix(path, "/whitelist/")
		s.handleDeleteWhitelist(w, r, entryID)
	// Org whitelist management (new)
	case path == "/org-whitelists" && r.Method == http.MethodGet:
		s.handleListAllOrgWhitelists(w, r)
	// App whitelist management (new)
	case path == "/app-whitelists" && r.Method == http.MethodGet:
		s.handleListAllAppWhitelists(w, r)

	// Tunnel management
	case path == "/tunnels" && r.Method == http.MethodGet:
		s.handleListTunnels(w, r)

	// Stats
	case path == "/stats" && r.Method == http.MethodGet:
		s.handleStats(w, r)

	// Organization management
	case path == "/organizations" && r.Method == http.MethodGet:
		s.handleListOrganizations(w, r)
	case path == "/organizations" && r.Method == http.MethodPost:
		s.handleCreateOrganization(w, r)
	case strings.HasPrefix(path, "/organizations/") && strings.HasSuffix(path, "/policy") && r.Method == http.MethodGet:
		orgID := strings.TrimSuffix(strings.TrimPrefix(path, "/organizations/"), "/policy")
		s.handleGetOrgPolicy(w, r, orgID)
	case strings.HasPrefix(path, "/organizations/") && strings.HasSuffix(path, "/policy") && r.Method == http.MethodPut:
		orgID := strings.TrimSuffix(strings.TrimPrefix(path, "/organizations/"), "/policy")
		s.handleSetOrgPolicy(w, r, orgID)
	case strings.HasPrefix(path, "/organizations/") && strings.HasSuffix(path, "/plan") && r.Method == http.MethodPut:
		orgID := strings.TrimSuffix(strings.TrimPrefix(path, "/organizations/"), "/plan")
		s.handleSetOrganizationPlan(w, r, orgID)
	case strings.HasPrefix(path, "/organizations/") && r.Method == http.MethodPut:
		orgID := strings.TrimPrefix(path, "/organizations/")
		s.handleUpdateOrganization(w, r, orgID)
	case strings.HasPrefix(path, "/organizations/") && r.Method == http.MethodDelete:
		orgID := strings.TrimPrefix(path, "/organizations/")
		s.handleDeleteOrganization(w, r, orgID)
	case strings.HasPrefix(path, "/organizations/") && r.Method == http.MethodGet:
		// GET /organizations/:id - single org (must come after /policy and /usage routes)
		orgID := strings.TrimPrefix(path, "/organizations/")
		if !strings.Contains(orgID, "/") {
			s.handleGetOrganization(w, r, orgID)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}

	// Application management
	case path == "/applications" && r.Method == http.MethodGet:
		s.handleListApplications(w, r)
	case path == "/applications" && r.Method == http.MethodPost:
		s.handleCreateApplication(w, r)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/stats") && r.Method == http.MethodGet:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/stats")
		s.handleGetApplicationStats(w, r, appID)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/tunnels") && r.Method == http.MethodGet:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/tunnels")
		s.handleGetApplicationTunnels(w, r, appID)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/policy") && r.Method == http.MethodGet:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/policy")
		s.handleGetAppPolicy(w, r, appID)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/policy") && r.Method == http.MethodPut:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/policy")
		s.handleSetAppPolicy(w, r, appID)
	case strings.HasPrefix(path, "/applications/") && r.Method == http.MethodGet:
		appID := strings.TrimPrefix(path, "/applications/")
		s.handleGetApplication(w, r, appID)
	case strings.HasPrefix(path, "/applications/") && r.Method == http.MethodPut:
		appID := strings.TrimPrefix(path, "/applications/")
		s.handleUpdateApplication(w, r, appID)
	case strings.HasPrefix(path, "/applications/") && r.Method == http.MethodDelete:
		appID := strings.TrimPrefix(path, "/applications/")
		s.handleDeleteApplication(w, r, appID)

	// API Key management
	case path == "/api-keys" && r.Method == http.MethodGet:
		s.handleListAPIKeys(w, r)
	case path == "/api-keys" && r.Method == http.MethodPost:
		s.handleCreateAPIKey(w, r)
	case strings.HasPrefix(path, "/api-keys/") && r.Method == http.MethodDelete:
		keyID := strings.TrimPrefix(path, "/api-keys/")
		s.handleDeleteAPIKey(w, r, keyID)

	// Audit log
	case path == "/audit" && r.Method == http.MethodGet:
		s.handleListAuditEvents(w, r)
	case path == "/audit/stats" && r.Method == http.MethodGet:
		s.handleAuditStats(w, r)

	// Plan management
	case path == "/plans" && r.Method == http.MethodGet:
		s.handleListPlans(w, r)
	case path == "/plans" && r.Method == http.MethodPost:
		s.handleCreatePlan(w, r)
	case strings.HasPrefix(path, "/plans/") && r.Method == http.MethodGet:
		planID := strings.TrimPrefix(path, "/plans/")
		s.handleGetPlan(w, r, planID)
	case strings.HasPrefix(path, "/plans/") && r.Method == http.MethodPut:
		planID := strings.TrimPrefix(path, "/plans/")
		s.handleUpdatePlan(w, r, planID)
	case strings.HasPrefix(path, "/plans/") && r.Method == http.MethodDelete:
		planID := strings.TrimPrefix(path, "/plans/")
		s.handleDeletePlan(w, r, planID)

	// Usage management
	case path == "/usage/summary" && r.Method == http.MethodGet:
		s.handleUsageSummary(w, r)
	case strings.HasPrefix(path, "/organizations/") && strings.HasSuffix(path, "/usage") && r.Method == http.MethodGet:
		orgID := strings.TrimSuffix(strings.TrimPrefix(path, "/organizations/"), "/usage")
		s.handleGetOrganizationUsage(w, r, orgID)

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

// ============================================
// Admin Self-Management Handlers
// ============================================

// handleAdminGetMe returns the current admin's account info
func (s *Server) handleAdminGetMe(w http.ResponseWriter, r *http.Request, admin *struct {
	ID       string
	Username string
	IsAdmin  bool
}) {
	account, err := s.db.GetAccountByID(admin.ID)
	if err != nil {
		log.Printf("Failed to get admin account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"account": map[string]interface{}{
			"id":          account.ID,
			"username":    account.Username,
			"isAdmin":     account.IsAdmin,
			"totpEnabled": account.TOTPEnabled,
			"createdAt":   account.CreatedAt,
			"lastUsed":    account.LastUsed,
			"hasPassword": account.PasswordHash != "",
		},
	})
}

// handleAdminSetMyPassword sets the current admin's password
func (s *Server) handleAdminSetMyPassword(w http.ResponseWriter, r *http.Request, admin *struct {
	ID       string
	Username string
	IsAdmin  bool
}) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password == "" || len(req.Password) < 8 {
		jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpdateAccountPassword(admin.ID, passwordHash); err != nil {
		log.Printf("Failed to set password: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Admin %s changed their password", admin.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleAdminGetMyTOTPSetup generates a new TOTP secret for the current admin
func (s *Server) handleAdminGetMyTOTPSetup(w http.ResponseWriter, r *http.Request, admin *struct {
	ID       string
	Username string
	IsAdmin  bool
}) {
	// Generate TOTP secret
	totpKey, err := auth.GenerateTOTPSecret(admin.Username)
	if err != nil {
		log.Printf("Failed to generate TOTP secret: %v", err)
		jsonError(w, "Failed to generate TOTP", http.StatusInternalServerError)
		return
	}

	// Encrypt and store the secret (but don't enable yet)
	encryptedSecret, err := auth.EncryptTOTPSecret(totpKey.Secret)
	if err != nil {
		log.Printf("Failed to encrypt TOTP secret: %v", err)
		jsonError(w, "Failed to setup TOTP", http.StatusInternalServerError)
		return
	}

	// Store the secret (not enabled until verified)
	if err := s.db.UpdateAccountTOTP(admin.ID, encryptedSecret, false); err != nil {
		log.Printf("Failed to store TOTP secret: %v", err)
		jsonError(w, "Failed to setup TOTP", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"secret":  totpKey.Secret,
		"url":     totpKey.URL,
	})
}

// handleAdminEnableMyTOTP verifies the TOTP code and enables TOTP for the current admin
func (s *Server) handleAdminEnableMyTOTP(w http.ResponseWriter, r *http.Request, admin *struct {
	ID       string
	Username string
	IsAdmin  bool
}) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" || len(req.Code) != 6 {
		jsonError(w, "Valid 6-digit code required", http.StatusBadRequest)
		return
	}

	// Get account
	account, err := s.db.GetAccountByID(admin.ID)
	if err != nil || account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.TOTPSecret == "" {
		jsonError(w, "TOTP setup not initiated. Call GET /admin/me/totp/setup first", http.StatusBadRequest)
		return
	}

	// Decrypt the TOTP secret
	secret, err := auth.DecryptTOTPSecret(account.TOTPSecret)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		jsonError(w, "Failed to verify TOTP", http.StatusInternalServerError)
		return
	}

	// Validate the code
	if !auth.ValidateTOTPWithWindow(secret, req.Code) {
		jsonError(w, "Invalid TOTP code", http.StatusUnauthorized)
		return
	}

	// Enable TOTP
	if err := s.db.UpdateAccountTOTP(admin.ID, account.TOTPSecret, true); err != nil {
		log.Printf("Failed to enable TOTP: %v", err)
		jsonError(w, "Failed to enable TOTP", http.StatusInternalServerError)
		return
	}

	log.Printf("TOTP enabled for admin: %s", admin.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleAdminDisableMyTOTP disables TOTP for the current admin (requires current TOTP code)
func (s *Server) handleAdminDisableMyTOTP(w http.ResponseWriter, r *http.Request, admin *struct {
	ID       string
	Username string
	IsAdmin  bool
}) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" || len(req.Code) != 6 {
		jsonError(w, "Valid 6-digit code required", http.StatusBadRequest)
		return
	}

	// Get account
	account, err := s.db.GetAccountByID(admin.ID)
	if err != nil || account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	if !account.TOTPEnabled || account.TOTPSecret == "" {
		jsonError(w, "TOTP is not enabled", http.StatusBadRequest)
		return
	}

	// Decrypt the TOTP secret
	secret, err := auth.DecryptTOTPSecret(account.TOTPSecret)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		jsonError(w, "Failed to verify TOTP", http.StatusInternalServerError)
		return
	}

	// Validate the code to authorize disabling
	if !auth.ValidateTOTPWithWindow(secret, req.Code) {
		jsonError(w, "Invalid TOTP code", http.StatusUnauthorized)
		return
	}

	// Disable TOTP
	if err := s.db.UpdateAccountTOTP(admin.ID, "", false); err != nil {
		log.Printf("Failed to disable TOTP: %v", err)
		jsonError(w, "Failed to disable TOTP", http.StatusInternalServerError)
		return
	}

	log.Printf("TOTP disabled for admin: %s", admin.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleListAccounts returns all accounts
func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.db.ListAccounts()
	if err != nil {
		log.Printf("Failed to list accounts: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Don't expose token hashes, but include org info
	result := make([]map[string]interface{}, len(accounts))
	for i, acc := range accounts {
		// Get org name if account has an org
		var orgName string
		if acc.OrgID != "" {
			if org, _ := s.db.GetOrganizationByID(acc.OrgID); org != nil {
				orgName = org.Name
			}
		}

		result[i] = map[string]interface{}{
			"id":          acc.ID,
			"username":    acc.Username,
			"isAdmin":     acc.IsAdmin,
			"isOrgAdmin":  acc.IsOrgAdmin,
			"totpEnabled": acc.TOTPEnabled,
			"createdAt":   acc.CreatedAt,
			"lastUsed":    acc.LastUsed,
			"active":      acc.Active,
			"orgId":       acc.OrgID,
			"orgName":     orgName,
			"hasPassword": acc.PasswordHash != "",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": result,
	})
}

// handleCreateAccount creates a new account
func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Username string `json:"username"`
		Password string `json:"password,omitempty"`
		IsAdmin  bool   `json:"isAdmin"`
		OrgID    string `json:"orgId,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		jsonError(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Validate password if provided
	if req.Password != "" && len(req.Password) < 8 {
		jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	existing, err := s.db.GetAccountByUsername(req.Username)
	if err != nil {
		log.Printf("Failed to check username: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		jsonError(w, "Username already exists", http.StatusConflict)
		return
	}

	// If org ID provided, verify it exists
	var orgName string
	if req.OrgID != "" {
		org, err := s.db.GetOrganizationByID(req.OrgID)
		if err != nil {
			log.Printf("Failed to check organization: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if org == nil {
			jsonError(w, "Organization not found", http.StatusNotFound)
			return
		}
		orgName = org.Name
	}

	// Generate token
	token, tokenHash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		passwordHash, err = auth.HashPassword(req.Password)
		if err != nil {
			log.Printf("Failed to hash password: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Create account
	var account *db.Account
	if passwordHash != "" {
		account, err = s.db.CreateAccountWithPassword(req.Username, tokenHash, passwordHash, req.IsAdmin)
	} else {
		account, err = s.db.CreateAccount(req.Username, tokenHash, req.IsAdmin)
	}
	if err != nil {
		log.Printf("Failed to create account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If org ID provided, link account to organization
	if req.OrgID != "" {
		if err := s.db.SetAccountOrganization(account.ID, req.OrgID); err != nil {
			log.Printf("Failed to link account to organization: %v", err)
			// Account created but org link failed - still return success but log the error
		} else {
			account.OrgID = req.OrgID
		}
	}

	log.Printf("Account created: %s (admin: %v, org: %s, hasPassword: %v)", req.Username, req.IsAdmin, req.OrgID, passwordHash != "")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"account": map[string]interface{}{
			"id":          account.ID,
			"username":    account.Username,
			"isAdmin":     account.IsAdmin,
			"createdAt":   account.CreatedAt,
			"orgId":       account.OrgID,
			"orgName":     orgName,
			"hasPassword": passwordHash != "",
		},
		"token": token, // Only returned once at creation
	})
}

// handleDeleteAccount deactivates an account
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if err := s.db.DeactivateAccount(accountID); err != nil {
		log.Printf("Failed to deactivate account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Account deactivated: %s", accountID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleActivateAccount reactivates an account
func (s *Server) handleActivateAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if err := s.db.ActivateAccount(accountID); err != nil {
		log.Printf("Failed to activate account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Account activated: %s", accountID)

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
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update account
	if err := s.db.UpdateAccountToken(accountID, tokenHash); err != nil {
		log.Printf("Failed to update token: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Token regenerated for account: %s", accountID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// handleSetAccountOrganization links or unlinks an account to/from an organization
func (s *Server) handleSetAccountOrganization(w http.ResponseWriter, r *http.Request, accountID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		OrgID string `json:"orgId"` // Empty string to unlink
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check account exists
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	var orgName string
	if req.OrgID != "" {
		// Verify organization exists
		org, err := s.db.GetOrganizationByID(req.OrgID)
		if err != nil {
			log.Printf("Failed to get organization: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if org == nil {
			jsonError(w, "Organization not found", http.StatusNotFound)
			return
		}
		orgName = org.Name
	}

	// Update account's organization
	if err := s.db.SetAccountOrganization(accountID, req.OrgID); err != nil {
		log.Printf("Failed to set account organization: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if req.OrgID == "" {
		log.Printf("Account %s unlinked from organization", accountID)
	} else {
		log.Printf("Account %s linked to organization %s", accountID, req.OrgID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"orgId":   req.OrgID,
		"orgName": orgName,
	})
}

// handleSetAccountPassword sets or updates the password for an account
func (s *Server) handleSetAccountPassword(w http.ResponseWriter, r *http.Request, accountID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		jsonError(w, "Password is required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check account exists
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Hash the new password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update account's password
	if err := s.db.UpdateAccountPassword(accountID, passwordHash); err != nil {
		log.Printf("Failed to set account password: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Password set for account %s", accountID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleGetAccount returns a single account by ID
func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Get org name if account has an org
	var orgName string
	if account.OrgID != "" {
		if org, _ := s.db.GetOrganizationByID(account.OrgID); org != nil {
			orgName = org.Name
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"account": map[string]interface{}{
			"id":          account.ID,
			"username":    account.Username,
			"isAdmin":     account.IsAdmin,
			"isOrgAdmin":  account.IsOrgAdmin,
			"totpEnabled": account.TOTPEnabled,
			"createdAt":   account.CreatedAt,
			"lastUsed":    account.LastUsed,
			"active":      account.Active,
			"orgId":       account.OrgID,
			"orgName":     orgName,
			"hasPassword": account.PasswordHash != "",
		},
	})
}

// handleSetAccountUsername updates the username for an account
func (s *Server) handleSetAccountUsername(w http.ResponseWriter, r *http.Request, accountID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		jsonError(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Check account exists
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Check if new username is already taken by another account
	existing, err := s.db.GetAccountByUsername(req.Username)
	if err != nil {
		log.Printf("Failed to check username: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil && existing.ID != accountID {
		jsonError(w, "Username already exists", http.StatusConflict)
		return
	}

	// Update username
	if err := s.db.UpdateAccountUsername(accountID, req.Username); err != nil {
		log.Printf("Failed to update username: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Username updated for account %s: %s", accountID, req.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": req.Username,
	})
}

// handleSetAccountOrgAdmin updates the org admin status for an account
func (s *Server) handleSetAccountOrgAdmin(w http.ResponseWriter, r *http.Request, accountID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		IsOrgAdmin bool `json:"isOrgAdmin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check account exists
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Only org users can be org admins
	if account.IsAdmin {
		jsonError(w, "System admins cannot be org admins", http.StatusBadRequest)
		return
	}
	if account.OrgID == "" {
		jsonError(w, "Account must belong to an organization", http.StatusBadRequest)
		return
	}

	// Update org admin status
	if err := s.db.UpdateAccountOrgAdmin(accountID, req.IsOrgAdmin); err != nil {
		log.Printf("Failed to update org admin status: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org admin status updated for account %s: %v", accountID, req.IsOrgAdmin)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"isOrgAdmin": req.IsOrgAdmin,
	})
}

// handleResetAccountTOTP disables TOTP for an account (admin only)
func (s *Server) handleResetAccountTOTP(w http.ResponseWriter, r *http.Request, accountID string) {
	// Check account exists
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Disable TOTP for the account
	if err := s.db.UpdateAccountTOTP(accountID, "", false); err != nil {
		log.Printf("Failed to reset TOTP: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("TOTP reset for account: %s (%s)", accountID, account.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleHardDeleteAccount permanently deletes an account
func (s *Server) handleHardDeleteAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	// Check account exists
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Permanently delete the account
	if err := s.db.HardDeleteAccount(accountID); err != nil {
		log.Printf("Failed to hard delete account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Account permanently deleted: %s (%s)", accountID, account.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
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

	// Ensure empty array instead of null
	if entries == nil {
		entries = []*db.WhitelistEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
	})
}

// handleAddWhitelist adds an IP range to the global whitelist
func (s *Server) handleAddWhitelist(w http.ResponseWriter, r *http.Request, createdBy string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		IPRange     string `json:"ipRange"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
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

// handleListAllOrgWhitelists returns all organization whitelist entries
func (s *Server) handleListAllOrgWhitelists(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.ListAllOrgWhitelists()
	if err != nil {
		log.Printf("Failed to list org whitelists: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure empty array instead of null
	if entries == nil {
		entries = []*db.OrgWhitelistEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
	})
}

// handleListAllAppWhitelists returns all application whitelist entries
func (s *Server) handleListAllAppWhitelists(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.ListAllAppWhitelists()
	if err != nil {
		log.Printf("Failed to list app whitelists: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure empty array instead of null
	if entries == nil {
		entries = []*db.AppWhitelistEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
	})
}

// handleListTunnels returns active tunnels
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	// Get in-memory active tunnels
	activeTunnels := s.GetActiveTunnels()

	// Ensure empty arrays instead of null
	if activeTunnels == nil {
		activeTunnels = []map[string]interface{}{}
	}

	// Get database tunnel records for additional info
	var dbTunnels interface{} = []interface{}{}
	if s.db != nil {
		if tunnels, err := s.db.ListActiveTunnels(); err == nil && tunnels != nil {
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

	jsonResponse(w, stats)
}

// ============================================
// Organization Management
// ============================================

// handleListOrganizations returns all organizations with app counts
func (s *Server) handleListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.db.ListOrganizations()
	if err != nil {
		log.Printf("Failed to list organizations: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Enrich with app counts, policy status, and plan info
	result := make([]map[string]interface{}, len(orgs))
	for i, org := range orgs {
		appCount, _ := s.db.CountApplicationsByOrg(org.ID)
		hasPolicy, _ := s.db.HasOrgAuthPolicy(org.ID)

		result[i] = map[string]interface{}{
			"id":        org.ID,
			"name":      org.Name,
			"createdAt": org.CreatedAt,
			"appCount":  appCount,
			"hasPolicy": hasPolicy,
		}

		// Add plan info if organization has a plan
		if org.PlanID != nil {
			result[i]["planId"] = *org.PlanID
			if plan, err := s.db.GetPlan(*org.PlanID); err == nil && plan != nil {
				result[i]["planName"] = plan.Name
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"organizations": result,
	})
}

// handleCreateOrganization creates a new organization
func (s *Server) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check if name already exists
	existing, err := s.db.GetOrganizationByName(req.Name)
	if err != nil {
		log.Printf("Failed to check organization name: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "Organization name already exists", http.StatusConflict)
		return
	}

	org, err := s.db.CreateOrganization(req.Name)
	if err != nil {
		log.Printf("Failed to create organization: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Organization created: %s", req.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"organization": org,
	})
}

// handleUpdateOrganization updates an organization
func (s *Server) handleUpdateOrganization(w http.ResponseWriter, r *http.Request, orgID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check org exists
	existing, err := s.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Printf("Failed to get organization: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	if err := s.db.UpdateOrganization(orgID, req.Name); err != nil {
		log.Printf("Failed to update organization: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Organization updated: %s -> %s", orgID, req.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleDeleteOrganization deletes an organization
func (s *Server) handleDeleteOrganization(w http.ResponseWriter, r *http.Request, orgID string) {
	// Check org exists
	existing, err := s.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Printf("Failed to get organization: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	// Check for dependent applications
	appCount, _ := s.db.CountApplicationsByOrg(orgID)
	if appCount > 0 {
		http.Error(w, "Cannot delete organization with applications", http.StatusConflict)
		return
	}

	// Delete org policy first
	s.db.DeleteOrgAuthPolicy(orgID)

	if err := s.db.DeleteOrganization(orgID); err != nil {
		log.Printf("Failed to delete organization: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Organization deleted: %s", orgID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleGetOrganization returns a single organization with full details
func (s *Server) handleGetOrganization(w http.ResponseWriter, r *http.Request, orgID string) {
	org, err := s.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Printf("Failed to get organization: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if org == nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	// Get enriched data
	appCount, _ := s.db.CountApplicationsByOrg(orgID)
	hasPolicy, _ := s.db.HasOrgAuthPolicy(orgID)
	accountCount, _ := s.db.CountAccountsByOrg(orgID)
	activeTunnels, _ := s.db.CountActiveTunnelsByOrg(orgID)

	result := map[string]interface{}{
		"id":            org.ID,
		"name":          org.Name,
		"createdAt":     org.CreatedAt,
		"appCount":      appCount,
		"hasPolicy":     hasPolicy,
		"accountCount":  accountCount,
		"activeTunnels": activeTunnels,
	}

	// Add plan info if set
	if org.PlanID != nil {
		result["planId"] = *org.PlanID
		if plan, err := s.db.GetPlan(*org.PlanID); err == nil && plan != nil {
			result["planName"] = plan.Name
			result["plan"] = plan
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetOrgPolicy returns the auth policy for an organization
func (s *Server) handleGetOrgPolicy(w http.ResponseWriter, r *http.Request, orgID string) {
	policy, err := s.db.GetOrgAuthPolicy(orgID)
	if err != nil {
		log.Printf("Failed to get org policy: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policy": policy,
	})
}

// handleSetOrgPolicy sets the auth policy for an organization
func (s *Server) handleSetOrgPolicy(w http.ResponseWriter, r *http.Request, orgID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		AuthType           string            `json:"authType"`
		BasicUsername      string            `json:"basicUsername,omitempty"`
		BasicPassword      string            `json:"basicPassword,omitempty"`
		OIDCIssuerURL      string            `json:"oidcIssuerUrl,omitempty"`
		OIDCClientID       string            `json:"oidcClientId,omitempty"`
		OIDCClientSecret   string            `json:"oidcClientSecret,omitempty"`
		OIDCScopes         []string          `json:"oidcScopes,omitempty"`
		OIDCAllowedDomains []string          `json:"oidcAllowedDomains,omitempty"`
		OIDCRequiredClaims map[string]string `json:"oidcRequiredClaims,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate auth type
	authType := db.AuthType(req.AuthType)
	if authType != db.AuthTypeBasic && authType != db.AuthTypeAPIKey && authType != db.AuthTypeOIDC {
		jsonError(w, "Invalid auth type", http.StatusBadRequest)
		return
	}

	policy := &db.OrgAuthPolicy{
		OrgID:    orgID,
		AuthType: authType,
	}

	switch authType {
	case db.AuthTypeBasic:
		if req.BasicUsername == "" || req.BasicPassword == "" {
			jsonError(w, "Basic auth requires username and password", http.StatusBadRequest)
			return
		}
		if len(req.BasicUsername) < 8 {
			jsonError(w, "Username must be at least 8 characters", http.StatusBadRequest)
			return
		}
		if len(req.BasicPassword) < 8 {
			jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		userHash, err := auth.HashPassword(req.BasicUsername)
		if err != nil {
			log.Printf("Failed to hash username: %v", err)
			jsonError(w, "Failed to hash username", http.StatusInternalServerError)
			return
		}
		passHash, err := auth.HashPassword(req.BasicPassword)
		if err != nil {
			log.Printf("Failed to hash password: %v", err)
			jsonError(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		policy.BasicUserHash = userHash
		policy.BasicPassHash = passHash

	case db.AuthTypeOIDC:
		if req.OIDCIssuerURL == "" || req.OIDCClientID == "" {
			jsonError(w, "OIDC requires issuer URL and client ID", http.StatusBadRequest)
			return
		}
		policy.OIDCIssuerURL = req.OIDCIssuerURL
		policy.OIDCClientID = req.OIDCClientID
		// Encrypt the OIDC client secret for secure storage
		if req.OIDCClientSecret != "" {
			encryptedSecret, err := auth.EncryptTOTPSecret(req.OIDCClientSecret)
			if err != nil {
				log.Printf("Failed to encrypt OIDC client secret: %v", err)
				jsonError(w, "Failed to encrypt client secret", http.StatusInternalServerError)
				return
			}
			policy.OIDCClientSecretEnc = encryptedSecret
		}
		policy.OIDCScopes = req.OIDCScopes
		policy.OIDCAllowedDomains = req.OIDCAllowedDomains
		policy.OIDCRequiredClaims = req.OIDCRequiredClaims
	}

	if err := s.db.CreateOrgAuthPolicy(policy); err != nil {
		log.Printf("Failed to set org policy: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org auth policy set: %s (%s)", orgID, authType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// Application Management
// ============================================

// handleListApplications returns all applications
func (s *Server) handleListApplications(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org")

	var apps []*db.Application
	var err error

	if orgID != "" {
		apps, err = s.db.ListApplicationsByOrg(orgID)
	} else {
		apps, err = s.db.ListAllApplications()
	}

	if err != nil {
		log.Printf("Failed to list applications: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Enrich with policy status, org name, active status and stats
	result := make([]map[string]interface{}, len(apps))
	for i, app := range apps {
		hasPolicy, _ := s.db.HasAppAuthPolicy(app.ID)

		// Get org name
		orgName := ""
		if org, _ := s.db.GetOrganizationByID(app.OrgID); org != nil {
			orgName = org.Name
		}

		// Get active tunnel count
		activeCount := s.GetActiveTunnelCountByApp(app.ID)

		// Get tunnel stats from database
		tunnelStats, _ := s.db.GetTunnelStatsByApp(app.ID)

		result[i] = map[string]interface{}{
			"id":                app.ID,
			"orgId":             app.OrgID,
			"orgName":           orgName,
			"subdomain":         app.Subdomain,
			"name":              app.Name,
			"authMode":          app.AuthMode,
			"authType":          app.AuthType,
			"createdAt":         app.CreatedAt,
			"hasPolicy":         hasPolicy,
			"isActive":          activeCount > 0,
			"activeTunnelCount": activeCount,
		}
		if tunnelStats != nil {
			result[i]["stats"] = tunnelStats
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"applications": result,
	})
}

// handleGetApplication returns a single application with stats
func (s *Server) handleGetApplication(w http.ResponseWriter, r *http.Request, appID string) {
	app, err := s.db.GetApplicationByID(appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	hasPolicy, _ := s.db.HasAppAuthPolicy(app.ID)

	// Get org name
	orgName := ""
	if org, _ := s.db.GetOrganizationByID(app.OrgID); org != nil {
		orgName = org.Name
	}

	// Get active tunnel count
	activeCount := s.GetActiveTunnelCountByApp(app.ID)

	// Get tunnel stats from database
	tunnelStats, _ := s.db.GetTunnelStatsByApp(app.ID)

	result := map[string]interface{}{
		"id":                app.ID,
		"orgId":             app.OrgID,
		"orgName":           orgName,
		"subdomain":         app.Subdomain,
		"name":              app.Name,
		"authMode":          app.AuthMode,
		"authType":          app.AuthType,
		"createdAt":         app.CreatedAt,
		"hasPolicy":         hasPolicy,
		"isActive":          activeCount > 0,
		"activeTunnelCount": activeCount,
	}
	if tunnelStats != nil {
		result["stats"] = tunnelStats
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"application": result,
	})
}

// handleGetApplicationStats returns stats for an application
func (s *Server) handleGetApplicationStats(w http.ResponseWriter, r *http.Request, appID string) {
	app, err := s.db.GetApplicationByID(appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	stats, err := s.db.GetTunnelStatsByApp(appID)
	if err != nil {
		log.Printf("Failed to get app stats: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get live active count from memory
	activeCount := s.GetActiveTunnelCountByApp(appID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"appId":             appID,
		"subdomain":         app.Subdomain,
		"activeTunnelCount": activeCount,
		"stats":             stats,
	})
}

// handleGetApplicationTunnels returns active tunnels for an application
func (s *Server) handleGetApplicationTunnels(w http.ResponseWriter, r *http.Request, appID string) {
	app, err := s.db.GetApplicationByID(appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	// Get live active tunnels from memory
	activeTunnels := s.GetActiveTunnelsByApp(appID)

	// Get database tunnel records
	var dbTunnels interface{}
	if tunnels, err := s.db.ListActiveTunnelsByApp(appID); err == nil {
		dbTunnels = tunnels
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active":  activeTunnels,
		"records": dbTunnels,
	})
}

// handleCreateApplication creates a new application
func (s *Server) handleCreateApplication(w http.ResponseWriter, r *http.Request) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		OrgID     string `json:"orgId"`
		Subdomain string `json:"subdomain"`
		Name      string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrgID == "" || req.Subdomain == "" {
		http.Error(w, "Organization ID and subdomain are required", http.StatusBadRequest)
		return
	}

	// Check org exists
	org, err := s.db.GetOrganizationByID(req.OrgID)
	if err != nil || org == nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	// Check subdomain availability
	available, err := s.db.IsSubdomainAvailable(req.Subdomain)
	if err != nil {
		log.Printf("Failed to check subdomain: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !available {
		http.Error(w, "Subdomain already in use", http.StatusConflict)
		return
	}

	app, err := s.db.CreateApplication(req.OrgID, req.Subdomain, req.Name)
	if err != nil {
		log.Printf("Failed to create application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Application created: %s (%s)", req.Subdomain, req.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"application": app,
	})
}

// handleUpdateApplication updates an application
func (s *Server) handleUpdateApplication(w http.ResponseWriter, r *http.Request, appID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		Name      string `json:"name"`
		Subdomain string `json:"subdomain,omitempty"`
		AuthMode  string `json:"authMode"`
		AuthType  string `json:"authType,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate auth mode
	authMode := db.AuthMode(req.AuthMode)
	if authMode != db.AuthModeInherit && authMode != db.AuthModeDisabled && authMode != db.AuthModeCustom {
		http.Error(w, "Invalid auth mode", http.StatusBadRequest)
		return
	}

	// Check app exists
	existing, err := s.db.GetApplicationByID(appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Use existing subdomain if not provided
	subdomain := req.Subdomain
	if subdomain == "" {
		subdomain = existing.Subdomain
	}

	authType := db.AuthType(req.AuthType)
	if err := s.db.UpdateApplicationFull(appID, req.Name, subdomain, authMode, authType); err != nil {
		log.Printf("Failed to update application: %v", err)
		if strings.Contains(err.Error(), "already in use") {
			jsonError(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Application updated: %s", appID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleDeleteApplication deletes an application
func (s *Server) handleDeleteApplication(w http.ResponseWriter, r *http.Request, appID string) {
	// Check app exists
	existing, err := s.db.GetApplicationByID(appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Application not found", http.StatusNotFound)
		return
	}

	// Delete app policy first
	s.db.DeleteAppAuthPolicy(appID)

	if err := s.db.DeleteApplication(appID); err != nil {
		log.Printf("Failed to delete application: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Application deleted: %s", appID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleGetAppPolicy returns the auth policy for an application
func (s *Server) handleGetAppPolicy(w http.ResponseWriter, r *http.Request, appID string) {
	policy, err := s.db.GetAppAuthPolicy(appID)
	if err != nil {
		log.Printf("Failed to get app policy: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policy": policy,
	})
}

// handleSetAppPolicy sets the auth policy for an application
func (s *Server) handleSetAppPolicy(w http.ResponseWriter, r *http.Request, appID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		AuthType           string            `json:"authType"`
		BasicUsername      string            `json:"basicUsername,omitempty"`
		BasicPassword      string            `json:"basicPassword,omitempty"`
		OIDCIssuerURL      string            `json:"oidcIssuerUrl,omitempty"`
		OIDCClientID       string            `json:"oidcClientId,omitempty"`
		OIDCClientSecret   string            `json:"oidcClientSecret,omitempty"`
		OIDCScopes         []string          `json:"oidcScopes,omitempty"`
		OIDCAllowedDomains []string          `json:"oidcAllowedDomains,omitempty"`
		OIDCRequiredClaims map[string]string `json:"oidcRequiredClaims,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate auth type
	authType := db.AuthType(req.AuthType)
	if authType != db.AuthTypeBasic && authType != db.AuthTypeAPIKey && authType != db.AuthTypeOIDC {
		jsonError(w, "Invalid auth type", http.StatusBadRequest)
		return
	}

	policy := &db.AppAuthPolicy{
		AppID:    appID,
		AuthType: authType,
	}

	switch authType {
	case db.AuthTypeBasic:
		if req.BasicUsername == "" || req.BasicPassword == "" {
			jsonError(w, "Basic auth requires username and password", http.StatusBadRequest)
			return
		}
		if len(req.BasicUsername) < 8 {
			jsonError(w, "Username must be at least 8 characters", http.StatusBadRequest)
			return
		}
		if len(req.BasicPassword) < 8 {
			jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		userHash, err := auth.HashPassword(req.BasicUsername)
		if err != nil {
			log.Printf("Failed to hash username: %v", err)
			jsonError(w, "Failed to hash username", http.StatusInternalServerError)
			return
		}
		passHash, err := auth.HashPassword(req.BasicPassword)
		if err != nil {
			log.Printf("Failed to hash password: %v", err)
			jsonError(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		policy.BasicUserHash = userHash
		policy.BasicPassHash = passHash

	case db.AuthTypeOIDC:
		if req.OIDCIssuerURL == "" || req.OIDCClientID == "" {
			jsonError(w, "OIDC requires issuer URL and client ID", http.StatusBadRequest)
			return
		}
		policy.OIDCIssuerURL = req.OIDCIssuerURL
		policy.OIDCClientID = req.OIDCClientID
		// Encrypt the OIDC client secret for secure storage
		if req.OIDCClientSecret != "" {
			encryptedSecret, err := auth.EncryptTOTPSecret(req.OIDCClientSecret)
			if err != nil {
				log.Printf("Failed to encrypt OIDC client secret: %v", err)
				jsonError(w, "Failed to encrypt client secret", http.StatusInternalServerError)
				return
			}
			policy.OIDCClientSecretEnc = encryptedSecret
		}
		policy.OIDCScopes = req.OIDCScopes
		policy.OIDCAllowedDomains = req.OIDCAllowedDomains
		policy.OIDCRequiredClaims = req.OIDCRequiredClaims
	}

	if err := s.db.CreateAppAuthPolicy(policy); err != nil {
		log.Printf("Failed to set app policy: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update app auth mode to custom
	s.db.UpdateApplicationAuthMode(appID, db.AuthModeCustom)

	log.Printf("App auth policy set: %s (%s)", appID, authType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// API Key Management
// ============================================

// handleListAPIKeys returns API keys filtered by org or app
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org")
	appID := r.URL.Query().Get("app")

	var keys []*db.APIKey
	var err error

	if appID != "" {
		keys, err = s.db.ListAPIKeysByApp(appID)
	} else if orgID != "" {
		keys, err = s.db.ListAPIKeysByOrg(orgID)
	} else {
		http.Error(w, "Either org or app parameter is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Failed to list API keys: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure empty array instead of null
	if keys == nil {
		keys = []*db.APIKey{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keys": keys,
	})
}

// handleCreateAPIKey creates a new API key
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var req struct {
		OrgID       string `json:"orgId,omitempty"`
		AppID       string `json:"appId,omitempty"`
		Description string `json:"description"`
		ExpiresIn   *int   `json:"expiresIn,omitempty"` // days
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrgID == "" {
		http.Error(w, "Organization ID is required", http.StatusBadRequest)
		return
	}

	var orgID, appID *string
	orgID = &req.OrgID
	if req.AppID != "" {
		appID = &req.AppID
	}

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &exp
	}

	rawKey, key, err := db.GenerateAPIKey(orgID, appID, req.Description, expiresAt)
	if err != nil {
		log.Printf("Failed to generate API key: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.db.CreateAPIKey(key); err != nil {
		log.Printf("Failed to create API key: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("API key created: %s... for org %s", key.KeyPrefix, req.OrgID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"key":     key,
		"rawKey":  rawKey, // Only returned once at creation
	})
}

// handleDeleteAPIKey revokes an API key
func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request, keyID string) {
	// Check key exists
	existing, err := s.db.GetAPIKeyByID(keyID)
	if err != nil {
		log.Printf("Failed to get API key: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeleteAPIKey(keyID); err != nil {
		log.Printf("Failed to delete API key: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("API key deleted: %s", keyID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// Audit Log
// ============================================

// handleListAuditEvents returns paginated audit events
func (s *Server) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var orgID, appID *string
	if v := query.Get("org"); v != "" {
		orgID = &v
	}
	if v := query.Get("app"); v != "" {
		appID = &v
	}

	limit := 50
	offset := 0
	if v := query.Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if v := query.Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	events, err := s.db.GetAuditEvents(orgID, appID, limit, offset)
	if err != nil {
		log.Printf("Failed to get audit events: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	total, _ := s.db.CountAuditEvents()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleAuditStats returns auth statistics
func (s *Server) handleAuditStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetAuthStats()
	if err != nil {
		log.Printf("Failed to get auth stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ============================================
// Plan Management
// ============================================

// handleListPlans returns all plans
func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.db.ListPlans()
	if err != nil {
		log.Printf("Failed to list plans: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"plans": plans,
	})
}

// handleCreatePlan creates a new plan
func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	var input db.CreatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		jsonError(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check for duplicate name
	existing, err := s.db.GetPlanByName(input.Name)
	if err != nil {
		log.Printf("Failed to check plan name: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		jsonError(w, "Plan name already exists", http.StatusConflict)
		return
	}

	plan, err := s.db.CreatePlan(input)
	if err != nil {
		log.Printf("Failed to create plan: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Plan created: %s (%s)", plan.Name, plan.ID)
	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, plan)
}

// handleGetPlan returns a specific plan
func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request, planID string) {
	plan, err := s.db.GetPlan(planID)
	if err != nil {
		log.Printf("Failed to get plan: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		jsonError(w, "Plan not found", http.StatusNotFound)
		return
	}

	// Get organizations using this plan
	orgs, err := s.db.GetOrganizationsUsingPlan(planID)
	if err != nil {
		log.Printf("Failed to get organizations using plan: %v", err)
	}

	jsonResponse(w, map[string]interface{}{
		"plan":          plan,
		"organizations": orgs,
	})
}

// handleUpdatePlan updates a plan
func (s *Server) handleUpdatePlan(w http.ResponseWriter, r *http.Request, planID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	// Check plan exists
	existing, err := s.db.GetPlan(planID)
	if err != nil {
		log.Printf("Failed to get plan: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		jsonError(w, "Plan not found", http.StatusNotFound)
		return
	}

	var input db.CreatePlanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		jsonError(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check for duplicate name (if changing)
	if input.Name != existing.Name {
		duplicate, err := s.db.GetPlanByName(input.Name)
		if err != nil {
			log.Printf("Failed to check plan name: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if duplicate != nil {
			jsonError(w, "Plan name already exists", http.StatusConflict)
			return
		}
	}

	plan, err := s.db.UpdatePlan(planID, input)
	if err != nil {
		log.Printf("Failed to update plan: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Plan updated: %s (%s)", plan.Name, plan.ID)
	jsonResponse(w, plan)
}

// handleDeletePlan deletes a plan
func (s *Server) handleDeletePlan(w http.ResponseWriter, r *http.Request, planID string) {
	// Check plan exists
	plan, err := s.db.GetPlan(planID)
	if err != nil {
		log.Printf("Failed to get plan: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		jsonError(w, "Plan not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeletePlan(planID); err != nil {
		log.Printf("Failed to delete plan: %v", err)
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}

	log.Printf("Plan deleted: %s (%s)", plan.Name, plan.ID)
	jsonResponse(w, map[string]bool{"success": true})
}

// ============================================
// Usage Management
// ============================================

// handleUsageSummary returns usage summary for all organizations
func (s *Server) handleUsageSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.db.GetUsageSummaryForAllOrgs()
	if err != nil {
		log.Printf("Failed to get usage summary: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get plans for enrichment
	plans, _ := s.db.ListPlans()
	planMap := make(map[string]*db.Plan)
	for _, p := range plans {
		planMap[p.ID] = p
	}

	// Enrich with plan details
	for _, item := range summary {
		if planID, ok := item["planId"].(string); ok {
			if plan, exists := planMap[planID]; exists {
				item["planName"] = plan.Name
				item["limits"] = map[string]interface{}{
					"bandwidthBytesMonthly": plan.BandwidthBytesMonthly,
					"tunnelHoursMonthly":    plan.TunnelHoursMonthly,
					"concurrentTunnelsMax":  plan.ConcurrentTunnelsMax,
					"requestsMonthly":       plan.RequestsMonthly,
				}
			}
		}
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	jsonResponse(w, map[string]interface{}{
		"organizations": summary,
		"periodStart":   periodStart,
		"periodEnd":     periodEnd,
	})
}

// handleGetOrganizationUsage returns usage for a specific organization
func (s *Server) handleGetOrganizationUsage(w http.ResponseWriter, r *http.Request, orgID string) {
	org, err := s.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Printf("Failed to get organization: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if org == nil {
		jsonError(w, "Organization not found", http.StatusNotFound)
		return
	}

	// Get current period usage
	usage, err := s.db.GetCurrentPeriodUsage(orgID)
	if err != nil {
		log.Printf("Failed to get usage: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get plan if set
	var plan *db.Plan
	if org.PlanID != nil {
		plan, _ = s.db.GetPlan(*org.PlanID)
	}

	// Get usage history based on query params
	query := r.URL.Query()
	periodType := db.PeriodType(query.Get("period"))
	if periodType == "" {
		periodType = db.PeriodDaily
	}

	days := 30
	if v := query.Get("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	end := time.Now()
	start := end.AddDate(0, 0, -days)
	history, err := s.db.GetUsageSnapshotsForOrg(orgID, periodType, start, end)
	if err != nil {
		log.Printf("Failed to get usage history: %v", err)
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	response := map[string]interface{}{
		"organization": org,
		"periodStart":  periodStart,
		"periodEnd":    periodEnd,
		"usage": map[string]interface{}{
			"bandwidthBytes":        usage.BandwidthBytes,
			"tunnelSeconds":         usage.TunnelSeconds,
			"requestCount":          usage.RequestCount,
			"peakConcurrentTunnels": usage.PeakConcurrentTunnels,
		},
		"history": history,
	}

	if plan != nil {
		response["plan"] = plan
		response["limits"] = map[string]interface{}{
			"bandwidthBytesMonthly": plan.BandwidthBytesMonthly,
			"tunnelHoursMonthly":    plan.TunnelHoursMonthly,
			"concurrentTunnelsMax":  plan.ConcurrentTunnelsMax,
			"requestsMonthly":       plan.RequestsMonthly,
		}
	}

	jsonResponse(w, response)
}

// handleSetOrganizationPlan sets the plan for an organization
func (s *Server) handleSetOrganizationPlan(w http.ResponseWriter, r *http.Request, orgID string) {
	if !validateJSONContentType(w, r) {
		return
	}
	limitRequestBody(r)

	org, err := s.db.GetOrganizationByID(orgID)
	if err != nil {
		log.Printf("Failed to get organization: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if org == nil {
		jsonError(w, "Organization not found", http.StatusNotFound)
		return
	}

	var input struct {
		PlanID *string `json:"planId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify plan exists if provided
	if input.PlanID != nil && *input.PlanID != "" {
		plan, err := s.db.GetPlan(*input.PlanID)
		if err != nil {
			log.Printf("Failed to get plan: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if plan == nil {
			jsonError(w, "Plan not found", http.StatusNotFound)
			return
		}
	}

	if err := s.db.UpdateOrganizationPlan(orgID, input.PlanID); err != nil {
		log.Printf("Failed to update organization plan: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Organization %s plan updated to: %v", orgID, input.PlanID)
	jsonResponse(w, map[string]bool{"success": true})
}
