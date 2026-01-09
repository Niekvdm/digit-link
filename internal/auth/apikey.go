package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/niekvdm/digit-link/internal/db"
	"github.com/niekvdm/digit-link/internal/policy"
)

// APIKeyAuthHandler handles API key authentication
type APIKeyAuthHandler struct {
	db *db.DB
}

// NewAPIKeyAuthHandler creates a new API key auth handler
func NewAPIKeyAuthHandler(database *db.DB) *APIKeyAuthHandler {
	return &APIKeyAuthHandler{db: database}
}

// Authenticate implements the AuthHandler interface for API key auth
func (h *APIKeyAuthHandler) Authenticate(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) *policy.AuthResult {
	// Extract API key from request
	apiKey := h.extractAPIKey(r)
	if apiKey == "" {
		return policy.Challenge("API key required")
	}

	// Validate API key
	key, err := h.db.ValidateAPIKey(apiKey)
	if err != nil {
		// Log error but don't expose details
		if ctx != nil {
			var orgID, appID *string
			if ctx.OrgID != "" {
				orgID = &ctx.OrgID
			}
			if ctx.AppID != "" {
				appID = &ctx.AppID
			}
			h.db.LogAuthFailure(orgID, appID, "api_key", GetClientIPFromRequest(r), "validation_error")
		}
		return policy.Failure("API key validation error")
	}

	if key == nil {
		// Invalid key
		if ctx != nil {
			var orgID, appID *string
			if ctx.OrgID != "" {
				orgID = &ctx.OrgID
			}
			if ctx.AppID != "" {
				appID = &ctx.AppID
			}
			h.db.LogAuthFailure(orgID, appID, "api_key", GetClientIPFromRequest(r), "invalid_key")
		}
		return policy.Failure("invalid API key")
	}

	// Check if key is for this org/app
	if ctx != nil {
		// If key is app-specific, it must match the app
		if key.AppID != nil {
			if ctx.AppID == "" || *key.AppID != ctx.AppID {
				// Key is for a different app - check if it's for the same org
				if key.OrgID == nil || ctx.OrgID == "" || *key.OrgID != ctx.OrgID {
					h.db.LogAuthFailure(&ctx.OrgID, &ctx.AppID, "api_key", GetClientIPFromRequest(r), "key_app_mismatch")
					return policy.Failure("API key not valid for this application")
				}
			}
		}

		// If key is org-specific (no app), it must match the org
		if key.AppID == nil && key.OrgID != nil {
			if ctx.OrgID == "" || *key.OrgID != ctx.OrgID {
				h.db.LogAuthFailure(&ctx.OrgID, &ctx.AppID, "api_key", GetClientIPFromRequest(r), "key_org_mismatch")
				return policy.Failure("API key not valid for this organization")
			}
		}
	}

	// Update last used timestamp
	h.db.UpdateAPIKeyLastUsed(key.ID)

	// Log successful authentication
	if ctx != nil {
		var orgID, appID *string
		if ctx.OrgID != "" {
			orgID = &ctx.OrgID
		}
		if ctx.AppID != "" {
			appID = &ctx.AppID
		}
		h.db.LogAuthSuccess(orgID, appID, "api_key", GetClientIPFromRequest(r), "api_key:"+key.KeyPrefix, key.ID)
	}

	return policy.SuccessWithKey(key.ID, key.KeyPrefix)
}

// Challenge sends an API key auth challenge to the client
func (h *APIKeyAuthHandler) Challenge(w http.ResponseWriter, r *http.Request, p *policy.EffectivePolicy, ctx *policy.AuthContext) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="digit-link", error="invalid_token"`)
	http.Error(w, "API key required. Provide via 'Authorization: Bearer <key>', 'X-API-Key: <key>', or 'X-Tunnel-API-Key: <key>' header", http.StatusUnauthorized)
}

// extractAPIKey extracts the API key from the request
func (h *APIKeyAuthHandler) extractAPIKey(r *http.Request) string {
	// Try X-API-Key header first
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Try X-Tunnel-API-Key header (alias for tunnel clients)
	apiKey = r.Header.Get("X-Tunnel-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Try Authorization: Bearer header (only if it's a digit-link API key)
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		// Only accept Bearer tokens that are digit-link API keys (dlk_ prefix)
		// This allows apps to use their own Bearer tokens without conflict
		if strings.HasPrefix(token, "dlk_") {
			return token
		}
	}

	// Try query parameter (less secure, but sometimes needed)
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return apiKey
	}

	return ""
}

// CreateAPIKeyForOrg creates a new API key for an organization
func CreateAPIKeyForOrg(database *db.DB, orgID, description string) (rawKey string, key *db.APIKey, err error) {
	rawKey, key, err = db.GenerateAPIKey(&orgID, nil, description, nil)
	if err != nil {
		return "", nil, err
	}

	err = database.CreateAPIKey(key)
	if err != nil {
		return "", nil, err
	}

	return rawKey, key, nil
}

// CreateAPIKeyForApp creates a new API key for an application
func CreateAPIKeyForApp(database *db.DB, orgID, appID, description string) (rawKey string, key *db.APIKey, err error) {
	rawKey, key, err = db.GenerateAPIKey(&orgID, &appID, description, nil)
	if err != nil {
		return "", nil, err
	}

	err = database.CreateAPIKey(key)
	if err != nil {
		return "", nil, err
	}

	return rawKey, key, nil
}

// RevokeAPIKey revokes (deletes) an API key
func RevokeAPIKey(database *db.DB, keyID string) error {
	return database.DeleteAPIKey(keyID)
}

// ListAPIKeysForOrg lists all API keys for an organization
func ListAPIKeysForOrg(database *db.DB, orgID string) ([]*db.APIKey, error) {
	return database.ListAPIKeysByOrg(orgID)
}

// ListAPIKeysForApp lists all API keys for an application
func ListAPIKeysForApp(database *db.DB, appID string) ([]*db.APIKey, error) {
	return database.ListAPIKeysByApp(appID)
}

// RotateAPIKey creates a new API key and optionally revokes the old one
// Returns the new raw key and key record
func RotateAPIKey(database *db.DB, oldKeyID string, revokeOld bool) (newRawKey string, newKey *db.APIKey, err error) {
	// Get the old key to copy its settings
	oldKey, err := database.GetAPIKeyByID(oldKeyID)
	if err != nil {
		return "", nil, err
	}
	if oldKey == nil {
		return "", nil, nil
	}

	// Create new key with same org/app association
	newRawKey, newKey, err = db.GenerateAPIKey(oldKey.OrgID, oldKey.AppID, oldKey.Description+" (rotated)", oldKey.ExpiresAt)
	if err != nil {
		return "", nil, err
	}

	err = database.CreateAPIKey(newKey)
	if err != nil {
		return "", nil, err
	}

	// Optionally revoke old key
	if revokeOld {
		database.DeleteAPIKey(oldKeyID)
	}

	return newRawKey, newKey, nil
}

// CreateAPIKeyWithExpiry creates an API key with a specific expiration time
func CreateAPIKeyWithExpiry(database *db.DB, orgID, appID *string, description string, expiresAt *time.Time) (rawKey string, key *db.APIKey, err error) {
	rawKey, key, err = db.GenerateAPIKey(orgID, appID, description, expiresAt)
	if err != nil {
		return "", nil, err
	}

	err = database.CreateAPIKey(key)
	if err != nil {
		return "", nil, err
	}

	return rawKey, key, nil
}

// CleanupExpiredKeys removes all expired API keys
func CleanupExpiredKeys(database *db.DB) (int64, error) {
	return database.DeleteExpiredAPIKeys()
}

// GetAPIKeyInfo gets information about an API key (without the hash)
func GetAPIKeyInfo(database *db.DB, keyID string) (*db.APIKey, error) {
	return database.GetAPIKeyByID(keyID)
}

// CountAPIKeysForOrg returns the number of API keys for an organization
func CountAPIKeysForOrg(database *db.DB, orgID string) (int, error) {
	return database.CountAPIKeysByOrg(orgID)
}
