package policy

import (
	"fmt"
	"time"

	"github.com/niekvdm/digit-link/internal/db"
)

// Resolver resolves effective authentication policies for requests
type Resolver struct {
	db *db.DB

	// secretDecryptor decrypts encrypted secrets (OIDC client secrets)
	// If nil, encrypted secrets cannot be decrypted
	secretDecryptor func(encrypted string) (string, error)

	// defaultDenyOnError determines behavior when policy cannot be loaded
	// If true (recommended), requests are denied on error
	// If false, requests are allowed on error (less secure)
	defaultDenyOnError bool
}

// ResolverOption is a functional option for configuring the resolver
type ResolverOption func(*Resolver)

// WithSecretDecryptor sets the secret decryptor function
func WithSecretDecryptor(decryptor func(string) (string, error)) ResolverOption {
	return func(r *Resolver) {
		r.secretDecryptor = decryptor
	}
}

// WithDefaultDenyOnError sets whether to deny on error (default: true)
func WithDefaultDenyOnError(deny bool) ResolverOption {
	return func(r *Resolver) {
		r.defaultDenyOnError = deny
	}
}

// NewResolver creates a new policy resolver
func NewResolver(database *db.DB, opts ...ResolverOption) *Resolver {
	r := &Resolver{
		db:                 database,
		defaultDenyOnError: true, // Fail closed by default
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveForSubdomain resolves the effective policy for a subdomain
// Returns nil if no authentication is required
func (r *Resolver) ResolveForSubdomain(subdomain string) (*EffectivePolicy, *AuthContext, error) {
	// First, check if this is a persistent application
	app, err := r.db.GetApplicationBySubdomain(subdomain)
	if err != nil {
		if r.defaultDenyOnError {
			return nil, nil, fmt.Errorf("failed to lookup application: %w", err)
		}
		return nil, &AuthContext{Subdomain: subdomain}, nil
	}

	ctx := &AuthContext{
		Subdomain: subdomain,
	}

	if app != nil {
		// This is a persistent application
		ctx.AppID = app.ID
		ctx.OrgID = app.OrgID
		ctx.App = app
		ctx.IsPersistentApp = true

		return r.resolveForApp(app, ctx)
	}

	// Not a persistent app - this is a random subdomain tunnel
	// We need to find the org from the active tunnel
	// For now, return nil (no auth) - the tunnel handler will need to provide org context
	return nil, ctx, nil
}

// ResolveForContext resolves the effective policy given a full auth context
// This is used when we already know the org/app (e.g., from tunnel registration)
func (r *Resolver) ResolveForContext(ctx *AuthContext) (*EffectivePolicy, error) {
	if ctx.App != nil {
		policy, _, err := r.resolveForApp(ctx.App, ctx)
		return policy, err
	}

	if ctx.OrgID != "" {
		return r.resolveForOrg(ctx.OrgID)
	}

	// No app or org context - no auth required
	return nil, nil
}

// resolveForApp resolves the effective policy for an application
func (r *Resolver) resolveForApp(app *db.Application, ctx *AuthContext) (*EffectivePolicy, *AuthContext, error) {
	switch db.AuthMode(app.AuthMode) {
	case db.AuthModeDisabled:
		// Auth explicitly disabled for this app
		return nil, ctx, nil

	case db.AuthModeCustom:
		// App has its own auth policy
		appPolicy, err := r.db.GetAppAuthPolicy(app.ID)
		if err != nil {
			if r.defaultDenyOnError {
				return nil, ctx, fmt.Errorf("failed to get app auth policy: %w", err)
			}
			return nil, ctx, nil
		}
		if appPolicy == nil {
			// Custom mode but no policy configured - fall back to org
			return r.resolveForOrgWithContext(app.OrgID, ctx)
		}
		policy, err := r.appPolicyToEffective(appPolicy, app.OrgID, app.ID)
		return policy, ctx, err

	case db.AuthModeInherit:
		fallthrough
	default:
		// Inherit from organization
		return r.resolveForOrgWithContext(app.OrgID, ctx)
	}
}

// resolveForOrg resolves the effective policy for an organization
func (r *Resolver) resolveForOrg(orgID string) (*EffectivePolicy, error) {
	policy, _, err := r.resolveForOrgWithContext(orgID, nil)
	return policy, err
}

// resolveForOrgWithContext resolves the org policy and returns updated context
func (r *Resolver) resolveForOrgWithContext(orgID string, ctx *AuthContext) (*EffectivePolicy, *AuthContext, error) {
	orgPolicy, err := r.db.GetOrgAuthPolicy(orgID)
	if err != nil {
		if r.defaultDenyOnError {
			return nil, ctx, fmt.Errorf("failed to get org auth policy: %w", err)
		}
		return nil, ctx, nil
	}

	if orgPolicy == nil {
		// No org policy configured - no auth required
		return nil, ctx, nil
	}

	policy, err := r.orgPolicyToEffective(orgPolicy)
	return policy, ctx, err
}

// orgPolicyToEffective converts an org auth policy to an effective policy
func (r *Resolver) orgPolicyToEffective(orgPolicy *db.OrgAuthPolicy) (*EffectivePolicy, error) {
	policy := &EffectivePolicy{
		Type:          AuthType(orgPolicy.AuthType),
		APIKeyEnabled: orgPolicy.APIKeyEnabled,
		OrgID:         orgPolicy.OrgID,
	}

	switch policy.Type {
	case AuthTypeBasic:
		policy.Basic = &BasicConfig{
			UserHash:        orgPolicy.BasicUserHash,
			PassHash:        orgPolicy.BasicPassHash,
			SessionDuration: time.Duration(orgPolicy.BasicSessionDuration) * time.Hour,
		}

	case AuthTypeAPIKey:
		policy.APIKey = &APIKeyConfig{}

	case AuthTypeOIDC:
		clientSecret := orgPolicy.OIDCClientSecretEnc
		if r.secretDecryptor != nil && clientSecret != "" {
			decrypted, err := r.secretDecryptor(clientSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt OIDC client secret: %w", err)
			}
			clientSecret = decrypted
		}

		policy.OIDC = &OIDCConfig{
			IssuerURL:      orgPolicy.OIDCIssuerURL,
			ClientID:       orgPolicy.OIDCClientID,
			ClientSecret:   clientSecret,
			Scopes:         orgPolicy.OIDCScopes,
			AllowedDomains: orgPolicy.OIDCAllowedDomains,
			RequiredClaims: orgPolicy.OIDCRequiredClaims,
		}
	}

	return policy, nil
}

// appPolicyToEffective converts an app auth policy to an effective policy
func (r *Resolver) appPolicyToEffective(appPolicy *db.AppAuthPolicy, orgID, appID string) (*EffectivePolicy, error) {
	policy := &EffectivePolicy{
		Type:          AuthType(appPolicy.AuthType),
		APIKeyEnabled: appPolicy.APIKeyEnabled,
		OrgID:         orgID,
		AppID:         appID,
	}

	switch policy.Type {
	case AuthTypeBasic:
		policy.Basic = &BasicConfig{
			UserHash:        appPolicy.BasicUserHash,
			PassHash:        appPolicy.BasicPassHash,
			SessionDuration: time.Duration(appPolicy.BasicSessionDuration) * time.Hour,
		}

	case AuthTypeAPIKey:
		policy.APIKey = &APIKeyConfig{}

	case AuthTypeOIDC:
		clientSecret := appPolicy.OIDCClientSecretEnc
		if r.secretDecryptor != nil && clientSecret != "" {
			decrypted, err := r.secretDecryptor(clientSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt OIDC client secret: %w", err)
			}
			clientSecret = decrypted
		}

		policy.OIDC = &OIDCConfig{
			IssuerURL:      appPolicy.OIDCIssuerURL,
			ClientID:       appPolicy.OIDCClientID,
			ClientSecret:   clientSecret,
			Scopes:         appPolicy.OIDCScopes,
			AllowedDomains: appPolicy.OIDCAllowedDomains,
			RequiredClaims: appPolicy.OIDCRequiredClaims,
		}
	}

	return policy, nil
}

// ResolveEffectivePolicy is a standalone function that resolves the effective policy
// given org and app policies (for use without database access)
func ResolveEffectivePolicy(orgPolicy *db.OrgAuthPolicy, app *db.Application, appPolicy *db.AppAuthPolicy) *EffectivePolicy {
	if app == nil {
		// No app - use org policy
		if orgPolicy == nil {
			return nil
		}
		return &EffectivePolicy{
			Type:  AuthType(orgPolicy.AuthType),
			OrgID: orgPolicy.OrgID,
		}
	}

	switch db.AuthMode(app.AuthMode) {
	case db.AuthModeDisabled:
		return nil

	case db.AuthModeCustom:
		if appPolicy == nil {
			// Fall back to org
			if orgPolicy == nil {
				return nil
			}
			return &EffectivePolicy{
				Type:  AuthType(orgPolicy.AuthType),
				OrgID: orgPolicy.OrgID,
			}
		}
		return &EffectivePolicy{
			Type:  AuthType(appPolicy.AuthType),
			OrgID: app.OrgID,
			AppID: app.ID,
		}

	case db.AuthModeInherit:
		fallthrough
	default:
		if orgPolicy == nil {
			return nil
		}
		return &EffectivePolicy{
			Type:  AuthType(orgPolicy.AuthType),
			OrgID: orgPolicy.OrgID,
		}
	}
}
