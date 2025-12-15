# API Reference - Tunnel Authentication

This document describes the programmatic API for managing tunnel-level authentication.

## Organizations

### Create Organization

```go
import "github.com/niekvdm/digit-link/internal/db"

org, err := database.CreateOrganization("my-organization")
```

### Get Organization

```go
// By ID
org, err := database.GetOrganizationByID(orgID)

// By name
org, err := database.GetOrganizationByName("my-organization")

// By account
org, err := database.GetOrganizationByAccountID(accountID)
```

### List Organizations

```go
orgs, err := database.ListOrganizations()
```

### Link Account to Organization

```go
err := database.SetAccountOrganization(accountID, orgID)
```

## Applications

### Create Application

```go
app, err := database.CreateApplication(orgID, "myapp", "My Application")
// Creates app with subdomain "myapp" and auth_mode "inherit"
```

### Get Application

```go
// By ID
app, err := database.GetApplicationByID(appID)

// By subdomain
app, err := database.GetApplicationBySubdomain("myapp")
```

### List Applications

```go
// All apps in an org
apps, err := database.ListApplicationsByOrg(orgID)

// All apps
apps, err := database.ListAllApplications()
```

### Update Auth Mode

```go
// Set to inherit, disabled, or custom
err := database.UpdateApplicationAuthMode(appID, db.AuthModeCustom)
```

### Check Subdomain Availability

```go
available, err := database.IsSubdomainAvailable("myapp")
```

## Auth Policies

### Organization Policy

```go
import (
    "github.com/niekvdm/digit-link/internal/db"
    "github.com/niekvdm/digit-link/internal/auth"
)

// Set Basic auth
err := auth.SetBasicAuthCredentials(database, orgID, "username", "password")

// Set OIDC
policy := &db.OrgAuthPolicy{
    OrgID:              orgID,
    AuthType:           db.AuthTypeOIDC,
    OIDCIssuerURL:      "https://accounts.google.com",
    OIDCClientID:       "client-id",
    OIDCClientSecretEnc: "client-secret",
    OIDCScopes:         []string{"openid", "email", "profile"},
    OIDCAllowedDomains: []string{"example.com"},
    OIDCRequiredClaims: map[string]string{"hd": "example.com"},
}
err := database.CreateOrgAuthPolicy(policy)

// Get policy
policy, err := database.GetOrgAuthPolicy(orgID)

// Delete policy
err := database.DeleteOrgAuthPolicy(orgID)
```

### Application Policy

```go
// Set custom Basic auth for app
err := auth.SetBasicAuthCredentialsForApp(database, appID, "admin", "secret")

// Set custom OIDC for app
policy := &db.AppAuthPolicy{
    AppID:    appID,
    AuthType: db.AuthTypeOIDC,
    // ... same fields as OrgAuthPolicy
}
err := database.CreateAppAuthPolicy(policy)

// Get policy
policy, err := database.GetAppAuthPolicy(appID)

// Delete policy
err := database.DeleteAppAuthPolicy(appID)
```

## API Keys

### Create API Key

```go
import "github.com/niekvdm/digit-link/internal/auth"

// Org-level key (works for all apps in org)
rawKey, key, err := auth.CreateAPIKeyForOrg(database, orgID, "CI/CD Pipeline")

// App-specific key
rawKey, key, err := auth.CreateAPIKeyForApp(database, orgID, appID, "App key")

// Key with expiration
expiry := time.Now().Add(30 * 24 * time.Hour)
rawKey, key, err := auth.CreateAPIKeyWithExpiry(database, &orgID, nil, "Temp key", &expiry)
```

**Important:** `rawKey` is only available at creation time. Store it securely.

### Validate API Key

```go
key, err := database.ValidateAPIKey(rawKey)
// Returns nil if invalid or expired
```

### List API Keys

```go
// By org
keys, err := auth.ListAPIKeysForOrg(database, orgID)

// By app
keys, err := auth.ListAPIKeysForApp(database, appID)
```

### Rotate API Key

```go
// Create new key and optionally revoke old one
newRawKey, newKey, err := auth.RotateAPIKey(database, oldKeyID, true)
```

### Revoke API Key

```go
err := auth.RevokeAPIKey(database, keyID)
```

### Cleanup Expired Keys

```go
deleted, err := auth.CleanupExpiredKeys(database)
```

## Sessions

### Create Session

```go
session, err := database.CreateSession(
    &appID,           // optional
    &orgID,           // optional
    "user@example.com",
    map[string]string{"role": "admin"},
    24 * time.Hour,   // duration
)
```

### Validate Session

```go
// Basic validation
session, err := database.ValidateSession(sessionID)

// Validate for specific app/org
session, err := database.ValidateSessionForApp(sessionID, &appID, &orgID)
```

### Extend Session

```go
err := database.ExtendSession(sessionID, 24 * time.Hour)
```

### Delete Session

```go
err := database.DeleteSession(sessionID)

// Delete all sessions for an app
err := database.DeleteSessionsByApp(appID)

// Delete all sessions for an org
err := database.DeleteSessionsByOrg(orgID)
```

### Cleanup Expired Sessions

```go
deleted, err := database.DeleteExpiredSessions()
```

## Rate Limiting

### Check Rate Limit

```go
import "github.com/niekvdm/digit-link/internal/auth"

limiter := auth.NewRateLimiter(database, auth.DefaultRateLimiterConfig())

key := auth.IPRateLimitKey(clientIP)
// or: auth.AppIPRateLimitKey(appID, clientIP)

allowed, retryAfter := limiter.Allow(key)
if !allowed {
    // Return 429 with Retry-After header
}
```

### Record Results

```go
// On successful auth (reduces rate limit counter)
limiter.RecordSuccess(key)

// On failed auth (persists state)
limiter.RecordFailure(key)
```

### Check Block Status

```go
blocked, retryAfter := limiter.IsBlocked(key)
```

### Reset Rate Limit

```go
limiter.Reset(key) // Admin use only
```

## Audit Logging

### Log Events

```go
// Automatic logging is built into auth handlers
// Manual logging:
err := database.LogAuthSuccess(&orgID, &appID, "api_key", clientIP, userIdentity, keyID)
err := database.LogAuthFailure(&orgID, &appID, "basic", clientIP, "invalid_password")
```

### Query Audit Log

```go
// Get events with filtering
events, err := database.GetAuditEvents(&orgID, &appID, 100, 0)

// Get recent events
events, err := database.GetRecentAuditEvents(time.Now().Add(-24*time.Hour), 100)

// Get failed attempts for an IP
count, err := database.GetFailedAuthAttempts(clientIP, time.Now().Add(-15*time.Minute))
```

### Get Statistics

```go
stats, err := database.GetAuthStats()
// stats.TotalAttempts, stats.SuccessCount, stats.FailureCount, etc.
```

### Cleanup Old Events

```go
deleted, err := database.DeleteOldAuditEvents(90 * 24 * time.Hour) // 90 days
```

## Policy Resolution

### Using the Resolver

```go
import "github.com/niekvdm/digit-link/internal/policy"

resolver := policy.NewResolver(database, 
    policy.WithDefaultDenyOnError(true),
    policy.WithSecretDecryptor(decryptFunc),
)

// Resolve by subdomain
effectivePolicy, authCtx, err := resolver.ResolveForSubdomain("myapp")

// Resolve by context
effectivePolicy, err := resolver.ResolveForContext(authCtx)
```

### Using the Loader (with caching)

```go
loader := policy.NewLoader(database, resolver,
    policy.WithCacheTTL(5 * time.Minute),
)

effectivePolicy, authCtx, err := loader.LoadForSubdomain("myapp")

// Invalidate cache
loader.InvalidateApp(appID)
loader.InvalidateOrg(orgID)
loader.InvalidateAll()
```

## Auth Handlers

### Basic Auth Handler

```go
import "github.com/niekvdm/digit-link/internal/auth"

handler := auth.NewBasicAuthHandler(database)
result := handler.Authenticate(w, r, effectivePolicy, authCtx)

if !result.Authenticated {
    handler.Challenge(w, r, effectivePolicy, authCtx)
    return
}
```

### API Key Handler

```go
handler := auth.NewAPIKeyAuthHandler(database)
result := handler.Authenticate(w, r, effectivePolicy, authCtx)
```

### OIDC Handler

```go
handler := auth.NewOIDCAuthHandler(database, "tunnel.digit.zone")

// In login endpoint
handler.HandleLogin(w, r, effectivePolicy, authCtx)

// In callback endpoint
handler.HandleCallback(w, r, effectivePolicy, authCtx)

// In logout endpoint
handler.HandleLogout(w, r)
```

## Auth Middleware

### Using the Middleware

```go
import "github.com/niekvdm/digit-link/internal/server"

middleware := server.NewAuthMiddleware(database,
    server.WithDefaultDeny(true),
    server.WithBasicHandler(basicHandler),
    server.WithAPIKeyHandler(apiKeyHandler),
    server.WithOIDCHandler(oidcHandler),
    server.WithRateLimiter(rateLimiter),
)

// In request handler
result, authCtx := middleware.AuthenticateRequest(w, r, subdomain)
if !middleware.HandleAuthResult(w, r, result, effectivePolicy, authCtx) {
    return // Auth failed, response already sent
}
// Continue with authenticated request
```

### Getting Auth Context from Request

```go
authResult := server.GetAuthResultFromContext(r)
authCtx := server.GetAuthContextFromContext(r)
policy := server.GetEffectivePolicyFromContext(r)
```
