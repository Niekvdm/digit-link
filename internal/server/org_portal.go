package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/niekvdm/digit-link/internal/auth"
	"github.com/niekvdm/digit-link/internal/db"
)

// handleOrg routes org portal API requests
func (s *Server) handleOrg(w http.ResponseWriter, r *http.Request) {
	// Verify org account authentication
	orgCtx, err := s.authenticateOrgAccount(r)
	if err != nil || orgCtx == nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Route org endpoints
	path := strings.TrimPrefix(r.URL.Path, "/org")

	switch {
	// Dashboard stats
	case path == "/stats" && r.Method == http.MethodGet:
		s.handleOrgStats(w, r, orgCtx)

	// Organization policy management
	case path == "/policy" && r.Method == http.MethodGet:
		s.handleOrgGetOrgPolicy(w, r, orgCtx)
	case path == "/policy" && r.Method == http.MethodPut:
		s.handleOrgSetOrgPolicy(w, r, orgCtx)

	// Application management
	case path == "/applications" && r.Method == http.MethodGet:
		s.handleOrgListApplications(w, r, orgCtx)
	case path == "/applications" && r.Method == http.MethodPost:
		s.handleOrgCreateApplication(w, r, orgCtx)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/stats") && r.Method == http.MethodGet:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/stats")
		s.handleOrgAppStats(w, r, orgCtx, appID)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/policy") && r.Method == http.MethodGet:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/policy")
		s.handleOrgGetAppPolicy(w, r, orgCtx, appID)
	case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/policy") && r.Method == http.MethodPut:
		appID := strings.TrimSuffix(strings.TrimPrefix(path, "/applications/"), "/policy")
		s.handleOrgSetAppPolicy(w, r, orgCtx, appID)
	case strings.HasPrefix(path, "/applications/") && r.Method == http.MethodGet:
		appID := strings.TrimPrefix(path, "/applications/")
		s.handleOrgGetApplication(w, r, orgCtx, appID)
	case strings.HasPrefix(path, "/applications/") && r.Method == http.MethodPut:
		appID := strings.TrimPrefix(path, "/applications/")
		s.handleOrgUpdateApplication(w, r, orgCtx, appID)
	case strings.HasPrefix(path, "/applications/") && r.Method == http.MethodDelete:
		appID := strings.TrimPrefix(path, "/applications/")
		s.handleOrgDeleteApplication(w, r, orgCtx, appID)

	// Whitelist management
	case path == "/whitelist" && r.Method == http.MethodGet:
		s.handleOrgListWhitelist(w, r, orgCtx)
	case path == "/whitelist" && r.Method == http.MethodPost:
		s.handleOrgAddWhitelist(w, r, orgCtx)
	case strings.HasPrefix(path, "/whitelist/") && r.Method == http.MethodDelete:
		entryID := strings.TrimPrefix(path, "/whitelist/")
		s.handleOrgDeleteWhitelist(w, r, orgCtx, entryID)
	case path == "/app-whitelist" && r.Method == http.MethodPost:
		s.handleOrgAddAppWhitelist(w, r, orgCtx)
	case strings.HasPrefix(path, "/app-whitelist/") && r.Method == http.MethodDelete:
		entryID := strings.TrimPrefix(path, "/app-whitelist/")
		s.handleOrgDeleteAppWhitelist(w, r, orgCtx, entryID)

	// API Key management
	case path == "/api-keys" && r.Method == http.MethodGet:
		s.handleOrgListAPIKeys(w, r, orgCtx)
	case path == "/api-keys" && r.Method == http.MethodPost:
		s.handleOrgCreateAPIKey(w, r, orgCtx)
	case strings.HasPrefix(path, "/api-keys/") && r.Method == http.MethodDelete:
		keyID := strings.TrimPrefix(path, "/api-keys/")
		s.handleOrgDeleteAPIKey(w, r, orgCtx, keyID)

	// Tunnel monitoring
	case path == "/tunnels" && r.Method == http.MethodGet:
		s.handleOrgListTunnels(w, r, orgCtx)

	// Account management (org admin only, except /accounts/me)
	case path == "/accounts/me" && r.Method == http.MethodGet:
		s.handleOrgGetMyAccount(w, r, orgCtx)
	case path == "/accounts/me" && r.Method == http.MethodPut:
		s.handleOrgUpdateMyAccount(w, r, orgCtx)
	case path == "/accounts/me/password" && r.Method == http.MethodPut:
		s.handleOrgSetMyPassword(w, r, orgCtx)
	case path == "/accounts/me/totp/setup" && r.Method == http.MethodGet:
		s.handleOrgGetMyTOTPSetup(w, r, orgCtx)
	case path == "/accounts/me/totp/setup" && r.Method == http.MethodPost:
		s.handleOrgEnableMyTOTP(w, r, orgCtx)
	case path == "/accounts/me/totp" && r.Method == http.MethodDelete:
		s.handleOrgDisableMyTOTP(w, r, orgCtx)
	case path == "/accounts" && r.Method == http.MethodGet:
		s.handleOrgListAccounts(w, r, orgCtx)
	case path == "/accounts" && r.Method == http.MethodPost:
		s.handleOrgCreateAccount(w, r, orgCtx)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/hard") && r.Method == http.MethodDelete:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/hard")
		s.handleOrgHardDeleteAccount(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/activate") && r.Method == http.MethodPost:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/activate")
		s.handleOrgActivateAccount(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/password") && r.Method == http.MethodPut:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/password")
		s.handleOrgSetAccountPassword(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/regenerate") && r.Method == http.MethodPost:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/regenerate")
		s.handleOrgRegenerateToken(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && strings.HasSuffix(path, "/org-admin") && r.Method == http.MethodPut:
		accountID := strings.TrimSuffix(strings.TrimPrefix(path, "/accounts/"), "/org-admin")
		s.handleOrgSetAccountOrgAdmin(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && r.Method == http.MethodGet:
		accountID := strings.TrimPrefix(path, "/accounts/")
		s.handleOrgGetAccount(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && r.Method == http.MethodPut:
		accountID := strings.TrimPrefix(path, "/accounts/")
		s.handleOrgUpdateAccount(w, r, orgCtx, accountID)
	case strings.HasPrefix(path, "/accounts/") && r.Method == http.MethodDelete:
		accountID := strings.TrimPrefix(path, "/accounts/")
		s.handleOrgDeactivateAccount(w, r, orgCtx, accountID)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// OrgContext holds authenticated org user context
type OrgContext struct {
	AccountID  string
	Username   string
	OrgID      string
	IsOrgAdmin bool
}

// authenticateOrgAccount verifies org account authentication from the request
func (s *Server) authenticateOrgAccount(r *http.Request) (*OrgContext, error) {
	if s.db == nil {
		return nil, nil
	}

	// Get token from header
	token := r.Header.Get("Authorization")
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}

	if token == "" {
		return nil, nil
	}

	// Validate as JWT token
	claims, err := auth.ValidateJWT(token)
	if err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, nil
	}

	// Org accounts must NOT be admin and MUST have org_id
	if claims.IsAdmin || claims.OrgID == "" {
		return nil, nil
	}

	// Get account to check org admin status
	account, err := s.db.GetAccountByID(claims.AccountID)
	if err != nil || account == nil {
		return nil, err
	}

	return &OrgContext{
		AccountID:  claims.AccountID,
		Username:   claims.Username,
		OrgID:      claims.OrgID,
		IsOrgAdmin: account.IsOrgAdmin,
	}, nil
}

// verifyOrgOwnership checks if an app belongs to the authenticated org
func (s *Server) verifyOrgOwnership(orgCtx *OrgContext, appID string) (*db.Application, error) {
	app, err := s.db.GetApplicationByID(appID)
	if err != nil {
		return nil, err
	}
	if app == nil || app.OrgID != orgCtx.OrgID {
		return nil, nil
	}
	return app, nil
}

// ============================================
// Stats
// ============================================

func (s *Server) handleOrgStats(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	stats := map[string]interface{}{
		"orgId": orgCtx.OrgID,
	}

	// Get application count
	if count, err := s.db.CountApplicationsByOrg(orgCtx.OrgID); err == nil {
		stats["applicationCount"] = count
	}

	// Get whitelist count
	if count, err := s.db.CountOrgWhitelist(orgCtx.OrgID); err == nil {
		stats["whitelistEntries"] = count
	}

	// Get tunnel stats
	if tunnelStats, err := s.db.GetTunnelStatsByOrg(orgCtx.OrgID); err == nil {
		stats["activeTunnels"] = tunnelStats.ActiveCount
		stats["totalConnections"] = tunnelStats.TotalConnections
		stats["totalBytesSent"] = tunnelStats.BytesSent
		stats["totalBytesReceived"] = tunnelStats.BytesReceived
	}

	// Get active tunnels from memory
	activeTunnels := s.GetActiveTunnelsByOrg(orgCtx.OrgID)
	stats["liveTunnels"] = len(activeTunnels)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ============================================
// Organization Policy
// ============================================

func (s *Server) handleOrgGetOrgPolicy(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	policy, err := s.db.GetOrgAuthPolicy(orgCtx.OrgID)
	if err != nil {
		log.Printf("Failed to get org policy: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policy": policy,
	})
}

func (s *Server) handleOrgSetOrgPolicy(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
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
		OrgID:    orgCtx.OrgID,
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

	log.Printf("Org auth policy set by user: %s (%s) for org %s", orgCtx.Username, authType, orgCtx.OrgID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// Applications
// ============================================

func (s *Server) handleOrgListApplications(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	apps, err := s.db.ListApplicationsByOrg(orgCtx.OrgID)
	if err != nil {
		log.Printf("Failed to list org applications: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Enrich with active status
	result := make([]map[string]interface{}, len(apps))
	for i, app := range apps {
		hasPolicy, _ := s.db.HasAppAuthPolicy(app.ID)
		activeCount := s.GetActiveTunnelCountByApp(app.ID)
		tunnelStats, _ := s.db.GetTunnelStatsByApp(app.ID)

		result[i] = map[string]interface{}{
			"id":                app.ID,
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

func (s *Server) handleOrgGetApplication(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, appID string) {
	app, err := s.verifyOrgOwnership(orgCtx, appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	hasPolicy, _ := s.db.HasAppAuthPolicy(app.ID)
	activeCount := s.GetActiveTunnelCountByApp(app.ID)
	tunnelStats, _ := s.db.GetTunnelStatsByApp(app.ID)

	result := map[string]interface{}{
		"id":                app.ID,
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

func (s *Server) handleOrgCreateApplication(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	var req struct {
		Subdomain string `json:"subdomain"`
		Name      string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Subdomain == "" {
		jsonError(w, "Subdomain is required", http.StatusBadRequest)
		return
	}

	// Check subdomain availability
	available, err := s.db.IsSubdomainAvailable(req.Subdomain)
	if err != nil {
		log.Printf("Failed to check subdomain: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !available {
		jsonError(w, "Subdomain already in use", http.StatusConflict)
		return
	}

	app, err := s.db.CreateApplication(orgCtx.OrgID, req.Subdomain, req.Name)
	if err != nil {
		log.Printf("Failed to create application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org application created: %s (%s) by %s", req.Subdomain, req.Name, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"application": app,
	})
}

func (s *Server) handleOrgUpdateApplication(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, appID string) {
	app, err := s.verifyOrgOwnership(orgCtx, appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name      string `json:"name"`
		Subdomain string `json:"subdomain"`
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
		jsonError(w, "Invalid auth mode", http.StatusBadRequest)
		return
	}

	// If subdomain is provided and different, use UpdateApplicationFull
	subdomain := req.Subdomain
	if subdomain == "" {
		subdomain = app.Subdomain
	}

	authType := db.AuthType(req.AuthType)
	if err := s.db.UpdateApplicationFull(appID, req.Name, subdomain, authMode, authType); err != nil {
		log.Printf("Failed to update application: %v", err)
		if strings.Contains(err.Error(), "already in use") {
			jsonError(w, err.Error(), http.StatusConflict)
		} else {
			jsonError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Org application updated: %s by %s", appID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleOrgDeleteApplication(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, appID string) {
	app, err := s.verifyOrgOwnership(orgCtx, appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	// Delete app policy first
	s.db.DeleteAppAuthPolicy(appID)

	if err := s.db.DeleteApplication(appID); err != nil {
		log.Printf("Failed to delete application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org application deleted: %s by %s", appID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleOrgAppStats(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, appID string) {
	app, err := s.verifyOrgOwnership(orgCtx, appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
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

	// Get live active count
	activeCount := s.GetActiveTunnelCountByApp(appID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"appId":             appID,
		"subdomain":         app.Subdomain,
		"activeTunnelCount": activeCount,
		"stats":             stats,
	})
}

func (s *Server) handleOrgGetAppPolicy(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, appID string) {
	app, err := s.verifyOrgOwnership(orgCtx, appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	policy, err := s.db.GetAppAuthPolicy(appID)
	if err != nil {
		log.Printf("Failed to get app policy: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policy": policy,
	})
}

func (s *Server) handleOrgSetAppPolicy(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, appID string) {
	app, err := s.verifyOrgOwnership(orgCtx, appID)
	if err != nil {
		log.Printf("Failed to get application: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

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

	log.Printf("Org app auth policy set: %s (%s) by %s", appID, authType, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// Whitelist
// ============================================

func (s *Server) handleOrgListWhitelist(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	// Get org whitelist
	orgEntries, err := s.db.ListOrgWhitelist(orgCtx.OrgID)
	if err != nil {
		log.Printf("Failed to list org whitelist: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure empty array instead of null
	if orgEntries == nil {
		orgEntries = []*db.OrgWhitelistEntry{}
	}

	// Get app whitelists for all apps in org
	apps, err := s.db.ListApplicationsByOrg(orgCtx.OrgID)
	if err != nil {
		log.Printf("Failed to list org apps: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	appWhitelists := make(map[string][]*db.AppWhitelistEntry)
	for _, app := range apps {
		entries, err := s.db.ListAppWhitelist(app.ID)
		if err != nil {
			continue
		}
		if len(entries) > 0 {
			appWhitelists[app.ID] = entries
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orgWhitelist":  orgEntries,
		"appWhitelists": appWhitelists,
	})
}

func (s *Server) handleOrgAddWhitelist(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	var req struct {
		IPRange     string `json:"ipRange"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.IPRange == "" {
		jsonError(w, "IP range is required", http.StatusBadRequest)
		return
	}

	entry, err := s.db.AddOrgWhitelist(orgCtx.OrgID, req.IPRange, req.Description, orgCtx.AccountID)
	if err != nil {
		log.Printf("Failed to add org whitelist entry: %v", err)
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Org whitelist entry added: %s (%s) by %s", req.IPRange, req.Description, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entry":   entry,
	})
}

func (s *Server) handleOrgDeleteWhitelist(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, entryID string) {
	// Verify ownership
	entry, err := s.db.GetOrgWhitelistEntry(entryID)
	if err != nil {
		log.Printf("Failed to get org whitelist entry: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if entry == nil || entry.OrgID != orgCtx.OrgID {
		jsonError(w, "Whitelist entry not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeleteOrgWhitelist(entryID); err != nil {
		log.Printf("Failed to delete org whitelist entry: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org whitelist entry deleted: %s by %s", entryID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleOrgAddAppWhitelist(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	var req struct {
		AppID       string `json:"appId"`
		IPRange     string `json:"ipRange"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AppID == "" || req.IPRange == "" {
		jsonError(w, "App ID and IP range are required", http.StatusBadRequest)
		return
	}

	// Verify app ownership
	app, err := s.verifyOrgOwnership(orgCtx, req.AppID)
	if err != nil || app == nil {
		jsonError(w, "Application not found", http.StatusNotFound)
		return
	}

	entry, err := s.db.AddAppWhitelist(req.AppID, req.IPRange, req.Description, orgCtx.AccountID)
	if err != nil {
		log.Printf("Failed to add app whitelist entry: %v", err)
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("App whitelist entry added: %s for %s (%s) by %s", req.IPRange, app.Subdomain, req.Description, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entry":   entry,
	})
}

func (s *Server) handleOrgDeleteAppWhitelist(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, entryID string) {
	// Get entry to verify ownership
	entry, err := s.db.GetAppWhitelistEntry(entryID)
	if err != nil {
		log.Printf("Failed to get app whitelist entry: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if entry == nil {
		jsonError(w, "Whitelist entry not found", http.StatusNotFound)
		return
	}

	// Verify app ownership
	app, err := s.verifyOrgOwnership(orgCtx, entry.AppID)
	if err != nil || app == nil {
		jsonError(w, "Whitelist entry not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeleteAppWhitelist(entryID); err != nil {
		log.Printf("Failed to delete app whitelist entry: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("App whitelist entry deleted: %s by %s", entryID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// API Keys
// ============================================

func (s *Server) handleOrgListAPIKeys(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	appID := r.URL.Query().Get("app")

	var keys []*db.APIKey
	var err error

	if appID != "" {
		// Verify app ownership
		app, err := s.verifyOrgOwnership(orgCtx, appID)
		if err != nil || app == nil {
			jsonError(w, "Application not found", http.StatusNotFound)
			return
		}
		keys, err = s.db.ListAPIKeysByApp(appID)
	} else {
		keys, err = s.db.ListAPIKeysByOrg(orgCtx.OrgID)
	}

	if err != nil {
		log.Printf("Failed to list API keys: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
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

func (s *Server) handleOrgCreateAPIKey(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	var req struct {
		AppID       string `json:"appId,omitempty"`
		Description string `json:"description"`
		ExpiresIn   *int   `json:"expiresIn,omitempty"` // days
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &exp
	}

	var rawKey string
	var key *db.APIKey
	var err error

	if req.AppID != "" {
		// Verify app ownership
		app, err := s.verifyOrgOwnership(orgCtx, req.AppID)
		if err != nil || app == nil {
			jsonError(w, "Application not found", http.StatusNotFound)
			return
		}
		// Create app-specific API key
		rawKey, key, err = db.GenerateAppAPIKey(orgCtx.OrgID, req.AppID, req.Description, expiresAt)
	} else {
		// Create org-level API key
		orgID := orgCtx.OrgID
		rawKey, key, err = db.GenerateAPIKey(&orgID, nil, req.Description, expiresAt)
	}

	if err != nil {
		log.Printf("Failed to generate API key: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.db.CreateAPIKey(key); err != nil {
		log.Printf("Failed to create API key: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org API key created: %s... by %s", key.KeyPrefix, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"key":     key,
		"rawKey":  rawKey,
	})
}

func (s *Server) handleOrgDeleteAPIKey(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, keyID string) {
	// Get key to verify ownership
	key, err := s.db.GetAPIKeyByID(keyID)
	if err != nil {
		log.Printf("Failed to get API key: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if key == nil {
		jsonError(w, "API key not found", http.StatusNotFound)
		return
	}

	// Verify ownership - key must belong to this org
	if key.OrgID == nil || *key.OrgID != orgCtx.OrgID {
		jsonError(w, "API key not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeleteAPIKey(keyID); err != nil {
		log.Printf("Failed to delete API key: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org API key deleted: %s by %s", keyID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// ============================================
// Tunnels
// ============================================

func (s *Server) handleOrgListTunnels(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	// Get live active tunnels from memory
	activeTunnels := s.GetActiveTunnelsByOrg(orgCtx.OrgID)

	// Ensure empty array instead of null
	if activeTunnels == nil {
		activeTunnels = []map[string]interface{}{}
	}

	// Get database tunnel records
	var dbTunnels interface{} = []interface{}{}
	if tunnels, err := s.db.ListActiveTunnelsByOrg(orgCtx.OrgID); err == nil && tunnels != nil {
		dbTunnels = tunnels
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active":  activeTunnels,
		"records": dbTunnels,
	})
}

// ============================================
// Helper methods on Server
// ============================================

// GetActiveTunnelsByOrg returns active tunnels for a specific organization
func (s *Server) GetActiveTunnelsByOrg(orgID string) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tunnels := make([]map[string]interface{}, 0)
	for subdomain, tunnel := range s.tunnels {
		if tunnel.OrgID == orgID {
			tunnels = append(tunnels, map[string]interface{}{
				"subdomain": subdomain,
				"url":       strings.Join([]string{s.scheme, "://", subdomain, ".", s.domain}, ""),
				"createdAt": tunnel.CreatedAt,
				"appId":     tunnel.AppID,
			})
		}
	}
	return tunnels
}

// GetActiveTunnelsByApp returns active tunnels for a specific application
func (s *Server) GetActiveTunnelsByApp(appID string) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tunnels := make([]map[string]interface{}, 0)
	for subdomain, tunnel := range s.tunnels {
		if tunnel.AppID == appID {
			tunnels = append(tunnels, map[string]interface{}{
				"subdomain": subdomain,
				"url":       strings.Join([]string{s.scheme, "://", subdomain, ".", s.domain}, ""),
				"createdAt": tunnel.CreatedAt,
			})
		}
	}
	return tunnels
}

// GetActiveTunnelCountByApp returns count of active tunnels for an app
func (s *Server) GetActiveTunnelCountByApp(appID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, tunnel := range s.tunnels {
		if tunnel.AppID == appID {
			count++
		}
	}
	return count
}

// GetActiveTunnelCountByOrg returns count of active tunnels for an org
func (s *Server) GetActiveTunnelCountByOrg(orgID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, tunnel := range s.tunnels {
		if tunnel.OrgID == orgID {
			count++
		}
	}
	return count
}

// ============================================
// Account Management (Org Portal)
// ============================================

// requireOrgAdmin checks if the user is an org admin
func (s *Server) requireOrgAdmin(w http.ResponseWriter, orgCtx *OrgContext) bool {
	if !orgCtx.IsOrgAdmin {
		jsonError(w, "Org admin access required", http.StatusForbidden)
		return false
	}
	return true
}

// verifyOrgAccountOwnership checks if an account belongs to the authenticated org
func (s *Server) verifyOrgAccountOwnership(orgCtx *OrgContext, accountID string) (*db.Account, error) {
	account, err := s.db.GetAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.OrgID != orgCtx.OrgID {
		return nil, nil
	}
	return account, nil
}

// handleOrgGetMyAccount returns the current user's account
func (s *Server) handleOrgGetMyAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	account, err := s.db.GetAccountByID(orgCtx.AccountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
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
			"isOrgAdmin":  account.IsOrgAdmin,
			"totpEnabled": account.TOTPEnabled,
			"createdAt":   account.CreatedAt,
			"lastUsed":    account.LastUsed,
			"active":      account.Active,
			"hasPassword": account.PasswordHash != "",
		},
	})
}

// handleOrgUpdateMyAccount updates the current user's account
func (s *Server) handleOrgUpdateMyAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username != "" {
		// Check if username is taken by another account
		existing, err := s.db.GetAccountByUsername(req.Username)
		if err != nil {
			log.Printf("Failed to check username: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if existing != nil && existing.ID != orgCtx.AccountID {
			jsonError(w, "Username already exists", http.StatusConflict)
			return
		}

		if err := s.db.UpdateAccountUsername(orgCtx.AccountID, req.Username); err != nil {
			log.Printf("Failed to update username: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Org user %s updated their account", orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgSetMyPassword sets the current user's password
func (s *Server) handleOrgSetMyPassword(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
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

	if err := s.db.UpdateAccountPassword(orgCtx.AccountID, passwordHash); err != nil {
		log.Printf("Failed to set password: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org user %s changed their password", orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgGetMyTOTPSetup generates a new TOTP secret for the current user
func (s *Server) handleOrgGetMyTOTPSetup(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	// Generate TOTP secret
	totpKey, err := auth.GenerateTOTPSecret(orgCtx.Username)
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
	if err := s.db.UpdateAccountTOTP(orgCtx.AccountID, encryptedSecret, false); err != nil {
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

// handleOrgEnableMyTOTP verifies the TOTP code and enables TOTP for the current user
func (s *Server) handleOrgEnableMyTOTP(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
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
	account, err := s.db.GetAccountByID(orgCtx.AccountID)
	if err != nil || account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.TOTPSecret == "" {
		jsonError(w, "TOTP setup not initiated. Call GET /accounts/me/totp/setup first", http.StatusBadRequest)
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
	if err := s.db.UpdateAccountTOTP(orgCtx.AccountID, account.TOTPSecret, true); err != nil {
		log.Printf("Failed to enable TOTP: %v", err)
		jsonError(w, "Failed to enable TOTP", http.StatusInternalServerError)
		return
	}

	log.Printf("TOTP enabled for org user: %s", orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgDisableMyTOTP disables TOTP for the current user (requires current TOTP code)
func (s *Server) handleOrgDisableMyTOTP(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
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
	account, err := s.db.GetAccountByID(orgCtx.AccountID)
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
	if err := s.db.UpdateAccountTOTP(orgCtx.AccountID, "", false); err != nil {
		log.Printf("Failed to disable TOTP: %v", err)
		jsonError(w, "Failed to disable TOTP", http.StatusInternalServerError)
		return
	}

	log.Printf("TOTP disabled for org user: %s", orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgListAccounts returns all accounts for the organization (org admin only)
func (s *Server) handleOrgListAccounts(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	accounts, err := s.db.ListAccountsByOrg(orgCtx.OrgID)
	if err != nil {
		log.Printf("Failed to list accounts: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(accounts))
	for i, acc := range accounts {
		result[i] = map[string]interface{}{
			"id":          acc.ID,
			"username":    acc.Username,
			"isOrgAdmin":  acc.IsOrgAdmin,
			"totpEnabled": acc.TOTPEnabled,
			"createdAt":   acc.CreatedAt,
			"lastUsed":    acc.LastUsed,
			"active":      acc.Active,
			"hasPassword": acc.PasswordHash != "",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": result,
	})
}

// handleOrgGetAccount returns a single account (org admin only)
func (s *Server) handleOrgGetAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
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
			"isOrgAdmin":  account.IsOrgAdmin,
			"totpEnabled": account.TOTPEnabled,
			"createdAt":   account.CreatedAt,
			"lastUsed":    account.LastUsed,
			"active":      account.Active,
			"hasPassword": account.PasswordHash != "",
		},
	})
}

// handleOrgCreateAccount creates a new account in the organization (org admin only)
func (s *Server) handleOrgCreateAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password,omitempty"`
		IsOrgAdmin bool   `json:"isOrgAdmin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		jsonError(w, "Username is required", http.StatusBadRequest)
		return
	}

	if req.Password != "" && len(req.Password) < 8 {
		jsonError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check if username exists
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
	account, err := s.db.CreateOrgAccountWithOrgAdmin(req.Username, tokenHash, passwordHash, orgCtx.OrgID, req.IsOrgAdmin)
	if err != nil {
		log.Printf("Failed to create account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org account created: %s by %s (isOrgAdmin: %v)", req.Username, orgCtx.Username, req.IsOrgAdmin)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"account": map[string]interface{}{
			"id":          account.ID,
			"username":    account.Username,
			"isOrgAdmin":  account.IsOrgAdmin,
			"createdAt":   account.CreatedAt,
			"hasPassword": passwordHash != "",
		},
		"token": token,
	})
}

// handleOrgUpdateAccount updates an account (org admin only)
func (s *Server) handleOrgUpdateAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username != "" {
		// Check if username is taken by another account
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

		if err := s.db.UpdateAccountUsername(accountID, req.Username); err != nil {
			log.Printf("Failed to update username: %v", err)
			jsonError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Org account %s updated by %s", accountID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgSetAccountPassword sets password for an account (org admin only)
func (s *Server) handleOrgSetAccountPassword(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

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

	if err := s.db.UpdateAccountPassword(accountID, passwordHash); err != nil {
		log.Printf("Failed to set password: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Password set for org account %s by %s", accountID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgRegenerateToken regenerates token for an account (org admin only)
func (s *Server) handleOrgRegenerateToken(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	// Generate new token
	token, tokenHash, err := auth.GenerateToken()
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpdateAccountToken(accountID, tokenHash); err != nil {
		log.Printf("Failed to update token: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Token regenerated for org account %s by %s", accountID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// handleOrgSetAccountOrgAdmin toggles org admin status (org admin only)
func (s *Server) handleOrgSetAccountOrgAdmin(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	// Cannot change own org admin status
	if accountID == orgCtx.AccountID {
		jsonError(w, "Cannot change your own org admin status", http.StatusBadRequest)
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	var req struct {
		IsOrgAdmin bool `json:"isOrgAdmin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateAccountOrgAdmin(accountID, req.IsOrgAdmin); err != nil {
		log.Printf("Failed to update org admin status: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org admin status updated for account %s to %v by %s", accountID, req.IsOrgAdmin, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"isOrgAdmin": req.IsOrgAdmin,
	})
}

// handleOrgActivateAccount activates an account (org admin only)
func (s *Server) handleOrgActivateAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := s.db.ActivateAccount(accountID); err != nil {
		log.Printf("Failed to activate account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org account %s activated by %s", accountID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgDeactivateAccount deactivates an account (org admin only)
func (s *Server) handleOrgDeactivateAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	// Cannot deactivate self
	if accountID == orgCtx.AccountID {
		jsonError(w, "Cannot deactivate your own account", http.StatusBadRequest)
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := s.db.DeactivateAccount(accountID); err != nil {
		log.Printf("Failed to deactivate account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org account %s deactivated by %s", accountID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// handleOrgHardDeleteAccount permanently deletes an account (org admin only)
func (s *Server) handleOrgHardDeleteAccount(w http.ResponseWriter, r *http.Request, orgCtx *OrgContext, accountID string) {
	if !s.requireOrgAdmin(w, orgCtx) {
		return
	}

	// Cannot delete self
	if accountID == orgCtx.AccountID {
		jsonError(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	account, err := s.verifyOrgAccountOwnership(orgCtx, accountID)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if account == nil {
		jsonError(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := s.db.HardDeleteAccount(accountID); err != nil {
		log.Printf("Failed to hard delete account: %v", err)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Org account %s permanently deleted by %s", accountID, orgCtx.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
