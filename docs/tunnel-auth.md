# Tunnel-Level Authentication

This document describes the tunnel-level authentication system that protects public traffic to tunneled applications.

## Overview

When a user accesses a tunnel URL (e.g., `https://myapp.tunnel.digit.zone`), the server can enforce authentication before forwarding requests to the tunnel origin. This provides an additional security layer on top of the existing tunnel client authentication.

## Architecture

```
┌─────────────┐     ┌──────────────────────────────────────────────┐     ┌────────────┐
│   Public    │     │              Edge Gateway                     │     │   Tunnel   │
│   User      │────▶│  TLS → Router → Policy → Auth → Forward      │────▶│   Client   │
└─────────────┘     └──────────────────────────────────────────────┘     └────────────┘
                                      │
                                      ▼
                              ┌───────────────┐
                              │   SQLite DB   │
                              │  - Policies   │
                              │  - Sessions   │
                              │  - API Keys   │
                              │  - Audit Log  │
                              └───────────────┘
```

## Key Concepts

### Organizations

Organizations are the top-level entity that groups accounts and applications. Each organization can have a default authentication policy that applies to all its applications.

### Applications

Applications are persistent entities with a reserved subdomain. Each application belongs to an organization and can have one of three authentication modes:

| Mode | Behavior |
|------|----------|
| `inherit` | Use the organization's default auth policy |
| `disabled` | No authentication required |
| `custom` | Use application-specific auth policy |

### Authentication Types

Three authentication methods are supported:

1. **Basic Auth** - Username/password with bcrypt hashing
2. **API Key** - Bearer token or X-API-Key/X-Tunnel-API-Key header with SHA-256 hashing
3. **OIDC** - OAuth2/OpenID Connect with any compatible provider

## Request Flow

1. User requests `https://myapp.tunnel.digit.zone/path`
2. Server extracts subdomain (`myapp`)
3. Server looks up application by subdomain
4. Policy resolver determines effective auth policy:
   - If app mode is `disabled` → skip auth
   - If app mode is `custom` → use app's policy
   - If app mode is `inherit` → use org's policy
5. Auth middleware enforces the policy
6. If authenticated → forward to tunnel
7. If not → challenge/redirect based on auth type

## Configuration

### Setting Up Organization Auth Policy

```go
// Create organization
org, _ := db.CreateOrganization("my-org")

// Set Basic auth for the organization
auth.SetBasicAuthCredentials(db, org.ID, "username", "password")

// Or set OIDC
policy := &db.OrgAuthPolicy{
    OrgID:              org.ID,
    AuthType:           db.AuthTypeOIDC,
    OIDCIssuerURL:      "https://accounts.google.com",
    OIDCClientID:       "your-client-id",
    OIDCClientSecretEnc: "your-client-secret",
    OIDCScopes:         []string{"openid", "email", "profile"},
    OIDCAllowedDomains: []string{"yourcompany.com"},
}
db.CreateOrgAuthPolicy(policy)
```

### Creating an Application

```go
// Create application with inherited auth
app, _ := db.CreateApplication(org.ID, "myapp", "My Application")

// Or create with custom auth
app, _ := db.CreateApplication(org.ID, "secure-app", "Secure App")
db.UpdateApplicationAuthMode(app.ID, db.AuthModeCustom)
auth.SetBasicAuthCredentialsForApp(db, app.ID, "admin", "secret")
```

### Creating API Keys

```go
// Create org-level API key
rawKey, key, _ := auth.CreateAPIKeyForOrg(db, org.ID, "CI/CD Pipeline")
// rawKey is shown once, key.KeyPrefix can be used to identify it later

// Create app-specific API key
rawKey, key, _ := auth.CreateAPIKeyForApp(db, org.ID, app.ID, "App-specific key")
```

## Auth Endpoints

For OIDC authentication, the following endpoints are available on each subdomain:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/__auth/login` | GET | Starts OIDC login flow |
| `/__auth/callback` | GET | OIDC callback (redirect from provider) |
| `/__auth/logout` | GET | Clears session and redirects |
| `/__auth/health` | GET | Auth system health check |

### Query Parameters

- `/__auth/login?redirect=/path` - Redirect to `/path` after successful login
- `/__auth/logout?redirect=/` - Redirect to `/` after logout

## Security Features

### Rate Limiting

Authentication attempts are rate-limited per IP address:

- **Window**: 15 minutes
- **Max Attempts**: 10 per window
- **Block Duration**: 30 minutes after exceeding limit

Rate limit state is persisted in SQLite to survive restarts.

### Audit Logging

All authentication events are logged to the `auth_audit_log` table:

- Timestamp
- Organization and Application IDs
- Auth type (basic/api_key/oidc)
- Success/failure status
- Failure reason (if applicable)
- Source IP
- User identity
- API key ID (if applicable)

### Security Headers

Auth endpoints include security headers:

- `Content-Security-Policy`
- `Strict-Transport-Security`
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`

### Session Security

OIDC sessions use:

- Cryptographically secure session IDs (32 bytes)
- HttpOnly cookies
- Secure flag (HTTPS only)
- SameSite=Lax
- 24-hour expiration (configurable)
- PKCE for OAuth2 flows

## Fail-Closed Behavior

The system is designed to fail closed:

- If policy cannot be loaded → deny request
- If OIDC provider is unreachable → deny request
- If session validation fails → redirect to login
- If rate limit state cannot be read → allow (degrade gracefully)

## Database Schema

See [Database Schema](./database-schema.md) for complete table definitions.
