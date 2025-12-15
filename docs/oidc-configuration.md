# OIDC Provider Configuration

This guide explains how to configure OIDC (OpenID Connect) authentication for tunnel-level auth.

## Supported Providers

Any OIDC-compliant provider should work, including:

- Google Workspace
- Microsoft Entra ID (Azure AD)
- Okta
- Auth0
- Keycloak
- OneLogin
- Authentik

## Configuration Steps

### 1. Create OAuth2 Application in Provider

Create an OAuth2/OIDC application in your identity provider with:

- **Application Type**: Web application
- **Redirect URI**: `https://<subdomain>.<your-domain>/__auth/callback`
  - Example: `https://myapp.tunnel.digit.zone/__auth/callback`
- **Scopes**: `openid`, `email`, `profile`

Note the **Client ID** and **Client Secret**.

### 2. Configure Organization Policy

```go
policy := &db.OrgAuthPolicy{
    OrgID:               orgID,
    AuthType:            db.AuthTypeOIDC,
    OIDCIssuerURL:       "https://your-provider.com",
    OIDCClientID:        "your-client-id",
    OIDCClientSecretEnc: "your-client-secret",
    OIDCScopes:          []string{"openid", "email", "profile"},
    OIDCAllowedDomains:  []string{"yourcompany.com"},
}
database.CreateOrgAuthPolicy(policy)
```

## Provider-Specific Examples

### Google Workspace

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create OAuth2 credentials
3. Configure consent screen
4. Set authorized redirect URIs

```go
policy := &db.OrgAuthPolicy{
    OrgID:               orgID,
    AuthType:            db.AuthTypeOIDC,
    OIDCIssuerURL:       "https://accounts.google.com",
    OIDCClientID:        "xxxx.apps.googleusercontent.com",
    OIDCClientSecretEnc: "GOCSPX-xxxxx",
    OIDCScopes:          []string{"openid", "email", "profile"},
    OIDCAllowedDomains:  []string{"yourcompany.com"},
    OIDCRequiredClaims:  map[string]string{"hd": "yourcompany.com"},
}
```

The `hd` claim ensures only users from your Google Workspace domain can authenticate.

### Microsoft Entra ID (Azure AD)

1. Go to [Azure Portal](https://portal.azure.com/) > Entra ID
2. App registrations > New registration
3. Add redirect URI as Web platform
4. Create client secret in Certificates & secrets

```go
policy := &db.OrgAuthPolicy{
    OrgID:               orgID,
    AuthType:            db.AuthTypeOIDC,
    OIDCIssuerURL:       "https://login.microsoftonline.com/{tenant-id}/v2.0",
    OIDCClientID:        "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    OIDCClientSecretEnc: "client-secret-value",
    OIDCScopes:          []string{"openid", "email", "profile"},
    OIDCAllowedDomains:  []string{"yourcompany.com"},
}
```

Replace `{tenant-id}` with your Azure AD tenant ID.

### Okta

1. Go to Okta Admin Console
2. Applications > Create App Integration
3. Select OIDC - Web Application
4. Configure redirect URIs

```go
policy := &db.OrgAuthPolicy{
    OrgID:               orgID,
    AuthType:            db.AuthTypeOIDC,
    OIDCIssuerURL:       "https://yourcompany.okta.com",
    OIDCClientID:        "0oaxxxxxxxxxx",
    OIDCClientSecretEnc: "client-secret",
    OIDCScopes:          []string{"openid", "email", "profile", "groups"},
}
```

### Auth0

1. Go to Auth0 Dashboard
2. Applications > Create Application
3. Select Regular Web Application
4. Configure callback URLs

```go
policy := &db.OrgAuthPolicy{
    OrgID:               orgID,
    AuthType:            db.AuthTypeOIDC,
    OIDCIssuerURL:       "https://yourcompany.auth0.com",
    OIDCClientID:        "xxxxxxxxxxxxx",
    OIDCClientSecretEnc: "xxxxxxxxxxxxx",
    OIDCScopes:          []string{"openid", "email", "profile"},
}
```

### Keycloak

```go
policy := &db.OrgAuthPolicy{
    OrgID:               orgID,
    AuthType:            db.AuthTypeOIDC,
    OIDCIssuerURL:       "https://keycloak.example.com/realms/your-realm",
    OIDCClientID:        "digit-link",
    OIDCClientSecretEnc: "client-secret",
    OIDCScopes:          []string{"openid", "email", "profile"},
}
```

## Claims-Based Authorization

### Email Domain Restriction

Restrict access to users with specific email domains:

```go
OIDCAllowedDomains: []string{"company.com", "subsidiary.com"}
```

### Custom Claims

Require specific claims to be present with certain values:

```go
OIDCRequiredClaims: map[string]string{
    "department": "engineering",
    "email_verified": "true",
}
```

For array claims (like groups), the required value must be present in the array:

```go
// Requires user to be in "admin" group
OIDCRequiredClaims: map[string]string{
    "groups": "admin",
}
```

## Multiple Redirect URIs

If you have multiple subdomains, you'll need to register redirect URIs for each:

```
https://app1.tunnel.digit.zone/__auth/callback
https://app2.tunnel.digit.zone/__auth/callback
https://app3.tunnel.digit.zone/__auth/callback
```

Some providers support wildcard redirect URIs (not recommended for production):

```
https://*.tunnel.digit.zone/__auth/callback
```

## Security Best Practices

1. **Use PKCE**: The system automatically uses PKCE (Proof Key for Code Exchange) for all OIDC flows, even with confidential clients.

2. **Secure Client Secrets**: Store client secrets encrypted. Consider using environment variables or a secret manager.

3. **Restrict Email Domains**: Always configure `OIDCAllowedDomains` to limit access to your organization's users.

4. **Use Required Claims**: Add additional claim requirements for sensitive applications.

5. **Short Session Duration**: Configure shorter session durations for sensitive apps.

6. **Regular Token Rotation**: OIDC tokens are short-lived; sessions handle persistent authentication.

## Troubleshooting

### Invalid Redirect URI

Ensure the redirect URI in your provider configuration exactly matches:
```
https://<subdomain>.<domain>/__auth/callback
```

### CORS Errors

The OIDC callback is a server-side redirect, not an API call, so CORS should not be an issue. If you see CORS errors, check your provider's configuration.

### Token Validation Fails

- Verify the issuer URL is correct
- Check that the client ID matches
- Ensure the provider's certificates are valid

### Email Domain Not Allowed

- Check `OIDCAllowedDomains` configuration
- Verify the user's email domain in the ID token
- Some providers don't include email in the ID token by default; add `email` scope

### Session Expired

- Sessions expire after 24 hours by default
- Users will be redirected to login again
- Extend session duration if needed
