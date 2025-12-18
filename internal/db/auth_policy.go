package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// OrgAuthPolicy represents an organization-level authentication policy
type OrgAuthPolicy struct {
	OrgID                string            `json:"orgId"`
	AuthType             AuthType          `json:"authType"`
	BasicUserHash        string            `json:"-"`
	BasicPassHash        string            `json:"-"`
	BasicSessionDuration int               `json:"basicSessionDuration,omitempty"` // Hours, 0 = default (24h)
	OIDCIssuerURL        string            `json:"oidcIssuerUrl,omitempty"`
	OIDCClientID         string            `json:"oidcClientId,omitempty"`
	OIDCClientSecretEnc  string            `json:"-"`
	OIDCScopes           []string          `json:"oidcScopes,omitempty"`
	OIDCAllowedDomains   []string          `json:"oidcAllowedDomains,omitempty"`
	OIDCRequiredClaims   map[string]string `json:"oidcRequiredClaims,omitempty"`
}

// AppAuthPolicy represents an application-level authentication policy
type AppAuthPolicy struct {
	AppID                string            `json:"appId"`
	AuthType             AuthType          `json:"authType"`
	BasicUserHash        string            `json:"-"`
	BasicPassHash        string            `json:"-"`
	BasicSessionDuration int               `json:"basicSessionDuration,omitempty"` // Hours, 0 = default (24h)
	OIDCIssuerURL        string            `json:"oidcIssuerUrl,omitempty"`
	OIDCClientID         string            `json:"oidcClientId,omitempty"`
	OIDCClientSecretEnc  string            `json:"-"`
	OIDCScopes           []string          `json:"oidcScopes,omitempty"`
	OIDCAllowedDomains   []string          `json:"oidcAllowedDomains,omitempty"`
	OIDCRequiredClaims   map[string]string `json:"oidcRequiredClaims,omitempty"`
}

// CreateOrgAuthPolicy creates or updates an organization auth policy
func (db *DB) CreateOrgAuthPolicy(policy *OrgAuthPolicy) error {
	scopesJSON, _ := json.Marshal(policy.OIDCScopes)
	domainsJSON, _ := json.Marshal(policy.OIDCAllowedDomains)
	claimsJSON, _ := json.Marshal(policy.OIDCRequiredClaims)

	_, err := db.conn.Exec(`
		INSERT INTO org_auth_policies (
			org_id, auth_type, basic_user_hash, basic_pass_hash, basic_session_duration,
			oidc_issuer_url, oidc_client_id, oidc_client_secret_enc,
			oidc_scopes, oidc_allowed_domains, oidc_required_claims
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(org_id) DO UPDATE SET
			auth_type = excluded.auth_type,
			basic_user_hash = excluded.basic_user_hash,
			basic_pass_hash = excluded.basic_pass_hash,
			basic_session_duration = excluded.basic_session_duration,
			oidc_issuer_url = excluded.oidc_issuer_url,
			oidc_client_id = excluded.oidc_client_id,
			oidc_client_secret_enc = excluded.oidc_client_secret_enc,
			oidc_scopes = excluded.oidc_scopes,
			oidc_allowed_domains = excluded.oidc_allowed_domains,
			oidc_required_claims = excluded.oidc_required_claims
	`, policy.OrgID, policy.AuthType, policy.BasicUserHash, policy.BasicPassHash, policy.BasicSessionDuration,
		policy.OIDCIssuerURL, policy.OIDCClientID, policy.OIDCClientSecretEnc,
		string(scopesJSON), string(domainsJSON), string(claimsJSON))

	if err != nil {
		return fmt.Errorf("failed to create org auth policy: %w", err)
	}
	return nil
}

// GetOrgAuthPolicy retrieves an organization auth policy
func (db *DB) GetOrgAuthPolicy(orgID string) (*OrgAuthPolicy, error) {
	policy := &OrgAuthPolicy{OrgID: orgID}
	var basicUserHash, basicPassHash, oidcIssuerURL, oidcClientID, oidcClientSecretEnc sql.NullString
	var basicSessionDuration sql.NullInt64
	var scopesJSON, domainsJSON, claimsJSON sql.NullString

	err := db.conn.QueryRow(`
		SELECT auth_type, basic_user_hash, basic_pass_hash, basic_session_duration,
			oidc_issuer_url, oidc_client_id, oidc_client_secret_enc,
			oidc_scopes, oidc_allowed_domains, oidc_required_claims
		FROM org_auth_policies WHERE org_id = ?
	`, orgID).Scan(
		&policy.AuthType, &basicUserHash, &basicPassHash, &basicSessionDuration,
		&oidcIssuerURL, &oidcClientID, &oidcClientSecretEnc,
		&scopesJSON, &domainsJSON, &claimsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get org auth policy: %w", err)
	}

	if basicUserHash.Valid {
		policy.BasicUserHash = basicUserHash.String
	}
	if basicPassHash.Valid {
		policy.BasicPassHash = basicPassHash.String
	}
	if basicSessionDuration.Valid {
		policy.BasicSessionDuration = int(basicSessionDuration.Int64)
	}
	if oidcIssuerURL.Valid {
		policy.OIDCIssuerURL = oidcIssuerURL.String
	}
	if oidcClientID.Valid {
		policy.OIDCClientID = oidcClientID.String
	}
	if oidcClientSecretEnc.Valid {
		policy.OIDCClientSecretEnc = oidcClientSecretEnc.String
	}
	if scopesJSON.Valid {
		json.Unmarshal([]byte(scopesJSON.String), &policy.OIDCScopes)
	}
	if domainsJSON.Valid {
		json.Unmarshal([]byte(domainsJSON.String), &policy.OIDCAllowedDomains)
	}
	if claimsJSON.Valid {
		json.Unmarshal([]byte(claimsJSON.String), &policy.OIDCRequiredClaims)
	}

	return policy, nil
}

// DeleteOrgAuthPolicy deletes an organization auth policy
func (db *DB) DeleteOrgAuthPolicy(orgID string) error {
	_, err := db.conn.Exec(`DELETE FROM org_auth_policies WHERE org_id = ?`, orgID)
	return err
}

// CreateAppAuthPolicy creates or updates an application auth policy
func (db *DB) CreateAppAuthPolicy(policy *AppAuthPolicy) error {
	scopesJSON, _ := json.Marshal(policy.OIDCScopes)
	domainsJSON, _ := json.Marshal(policy.OIDCAllowedDomains)
	claimsJSON, _ := json.Marshal(policy.OIDCRequiredClaims)

	_, err := db.conn.Exec(`
		INSERT INTO app_auth_policies (
			app_id, auth_type, basic_user_hash, basic_pass_hash, basic_session_duration,
			oidc_issuer_url, oidc_client_id, oidc_client_secret_enc,
			oidc_scopes, oidc_allowed_domains, oidc_required_claims
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(app_id) DO UPDATE SET
			auth_type = excluded.auth_type,
			basic_user_hash = excluded.basic_user_hash,
			basic_pass_hash = excluded.basic_pass_hash,
			basic_session_duration = excluded.basic_session_duration,
			oidc_issuer_url = excluded.oidc_issuer_url,
			oidc_client_id = excluded.oidc_client_id,
			oidc_client_secret_enc = excluded.oidc_client_secret_enc,
			oidc_scopes = excluded.oidc_scopes,
			oidc_allowed_domains = excluded.oidc_allowed_domains,
			oidc_required_claims = excluded.oidc_required_claims
	`, policy.AppID, policy.AuthType, policy.BasicUserHash, policy.BasicPassHash, policy.BasicSessionDuration,
		policy.OIDCIssuerURL, policy.OIDCClientID, policy.OIDCClientSecretEnc,
		string(scopesJSON), string(domainsJSON), string(claimsJSON))

	if err != nil {
		return fmt.Errorf("failed to create app auth policy: %w", err)
	}
	return nil
}

// GetAppAuthPolicy retrieves an application auth policy
func (db *DB) GetAppAuthPolicy(appID string) (*AppAuthPolicy, error) {
	policy := &AppAuthPolicy{AppID: appID}
	var basicUserHash, basicPassHash, oidcIssuerURL, oidcClientID, oidcClientSecretEnc sql.NullString
	var basicSessionDuration sql.NullInt64
	var scopesJSON, domainsJSON, claimsJSON sql.NullString

	err := db.conn.QueryRow(`
		SELECT auth_type, basic_user_hash, basic_pass_hash, basic_session_duration,
			oidc_issuer_url, oidc_client_id, oidc_client_secret_enc,
			oidc_scopes, oidc_allowed_domains, oidc_required_claims
		FROM app_auth_policies WHERE app_id = ?
	`, appID).Scan(
		&policy.AuthType, &basicUserHash, &basicPassHash, &basicSessionDuration,
		&oidcIssuerURL, &oidcClientID, &oidcClientSecretEnc,
		&scopesJSON, &domainsJSON, &claimsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get app auth policy: %w", err)
	}

	if basicUserHash.Valid {
		policy.BasicUserHash = basicUserHash.String
	}
	if basicPassHash.Valid {
		policy.BasicPassHash = basicPassHash.String
	}
	if basicSessionDuration.Valid {
		policy.BasicSessionDuration = int(basicSessionDuration.Int64)
	}
	if oidcIssuerURL.Valid {
		policy.OIDCIssuerURL = oidcIssuerURL.String
	}
	if oidcClientID.Valid {
		policy.OIDCClientID = oidcClientID.String
	}
	if oidcClientSecretEnc.Valid {
		policy.OIDCClientSecretEnc = oidcClientSecretEnc.String
	}
	if scopesJSON.Valid {
		json.Unmarshal([]byte(scopesJSON.String), &policy.OIDCScopes)
	}
	if domainsJSON.Valid {
		json.Unmarshal([]byte(domainsJSON.String), &policy.OIDCAllowedDomains)
	}
	if claimsJSON.Valid {
		json.Unmarshal([]byte(claimsJSON.String), &policy.OIDCRequiredClaims)
	}

	return policy, nil
}

// DeleteAppAuthPolicy deletes an application auth policy
func (db *DB) DeleteAppAuthPolicy(appID string) error {
	_, err := db.conn.Exec(`DELETE FROM app_auth_policies WHERE app_id = ?`, appID)
	return err
}

// HasOrgAuthPolicy checks if an organization has an auth policy configured
func (db *DB) HasOrgAuthPolicy(orgID string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM org_auth_policies WHERE org_id = ?
	`, orgID).Scan(&count)
	return count > 0, err
}

// HasAppAuthPolicy checks if an application has a custom auth policy
func (db *DB) HasAppAuthPolicy(appID string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM app_auth_policies WHERE app_id = ?
	`, appID).Scan(&count)
	return count > 0, err
}

// UpdateOrgBasicCredentials updates just the basic auth credentials for an org
func (db *DB) UpdateOrgBasicCredentials(orgID, userHash, passHash string) error {
	// Check if policy exists
	exists, err := db.HasOrgAuthPolicy(orgID)
	if err != nil {
		return err
	}

	if exists {
		_, err = db.conn.Exec(`
			UPDATE org_auth_policies 
			SET auth_type = ?, basic_user_hash = ?, basic_pass_hash = ?
			WHERE org_id = ?
		`, AuthTypeBasic, userHash, passHash, orgID)
	} else {
		_, err = db.conn.Exec(`
			INSERT INTO org_auth_policies (org_id, auth_type, basic_user_hash, basic_pass_hash)
			VALUES (?, ?, ?, ?)
		`, orgID, AuthTypeBasic, userHash, passHash)
	}
	return err
}

// UpdateAppBasicCredentials updates just the basic auth credentials for an app
func (db *DB) UpdateAppBasicCredentials(appID, userHash, passHash string) error {
	// Check if policy exists
	exists, err := db.HasAppAuthPolicy(appID)
	if err != nil {
		return err
	}

	if exists {
		_, err = db.conn.Exec(`
			UPDATE app_auth_policies 
			SET auth_type = ?, basic_user_hash = ?, basic_pass_hash = ?
			WHERE app_id = ?
		`, AuthTypeBasic, userHash, passHash, appID)
	} else {
		_, err = db.conn.Exec(`
			INSERT INTO app_auth_policies (app_id, auth_type, basic_user_hash, basic_pass_hash)
			VALUES (?, ?, ?, ?)
		`, appID, AuthTypeBasic, userHash, passHash)
	}
	return err
}

// ClearOrgBasicCredentials removes basic auth credentials from an org policy
func (db *DB) ClearOrgBasicCredentials(orgID string) error {
	_, err := db.conn.Exec(`
		UPDATE org_auth_policies 
		SET basic_user_hash = NULL, basic_pass_hash = NULL
		WHERE org_id = ?
	`, orgID)
	return err
}

// ClearAppBasicCredentials removes basic auth credentials from an app policy
func (db *DB) ClearAppBasicCredentials(appID string) error {
	_, err := db.conn.Exec(`
		UPDATE app_auth_policies 
		SET basic_user_hash = NULL, basic_pass_hash = NULL
		WHERE app_id = ?
	`, appID)
	return err
}
