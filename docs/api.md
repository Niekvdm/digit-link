# digit-link API Reference

## Overview

digit-link exposes three API groups:

1. **Admin API** (`/admin/*`) - System administration
2. **Org Portal API** (`/org/*`) - Organization self-service
3. **Auth API** (`/auth/*`) - Authentication endpoints
4. **Tunnel Auth API** (`/__auth/*`) - Per-subdomain SSO

## Authentication

### Admin API

Requires either:
- `X-Admin-Token: <api-token>` header, OR
- `Authorization: Bearer <jwt-token>` header

### Org Portal API

Requires:
- `Authorization: Bearer <jwt-token>` header

JWT tokens are obtained via the `/auth/login` endpoint.

---

## Admin API Endpoints

### Self-Management

#### GET `/admin/me`
Returns the current admin's account information.

**Response:**
```json
{
  "account": {
    "id": "uuid",
    "username": "admin",
    "isAdmin": true,
    "totpEnabled": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "lastUsed": "2024-01-15T12:00:00Z",
    "hasPassword": true
  }
}
```

#### PUT `/admin/me/password`
Set the current admin's password.

**Request:**
```json
{
  "password": "newPassword123"
}
```

**Response:**
```json
{
  "success": true
}
```

#### GET `/admin/me/totp/setup`
Generate a new TOTP secret for the current admin.

**Response:**
```json
{
  "success": true,
  "secret": "JBSWY3DPEHPK3PXP",
  "url": "otpauth://totp/digit-link:admin?secret=JBSWY3DPEHPK3PXP&issuer=digit-link"
}
```

#### POST `/admin/me/totp/setup`
Enable TOTP after verifying the code.

**Request:**
```json
{
  "code": "123456"
}
```

**Response:**
```json
{
  "success": true
}
```

#### DELETE `/admin/me/totp`
Disable TOTP (requires current code).

**Request:**
```json
{
  "code": "123456"
}
```

---

### Account Management

#### GET `/admin/accounts`
List all accounts.

**Response:**
```json
{
  "accounts": [
    {
      "id": "uuid",
      "username": "admin",
      "isAdmin": true,
      "isOrgAdmin": false,
      "totpEnabled": true,
      "createdAt": "2024-01-01T00:00:00Z",
      "lastUsed": "2024-01-15T12:00:00Z",
      "active": true,
      "orgId": "",
      "orgName": "",
      "hasPassword": true
    }
  ]
}
```

#### POST `/admin/accounts`
Create a new account.

**Request:**
```json
{
  "username": "newuser",
  "password": "optional-password",
  "isAdmin": false,
  "orgId": "org-uuid"
}
```

**Response:**
```json
{
  "success": true,
  "account": {
    "id": "new-uuid",
    "username": "newuser",
    "isAdmin": false,
    "createdAt": "2024-01-15T12:00:00Z",
    "orgId": "org-uuid",
    "orgName": "My Organization",
    "hasPassword": true
  },
  "token": "base64-encoded-token"
}
```

> ⚠️ The `token` is only returned once at creation time.

#### GET `/admin/accounts/{id}`
Get a single account by ID.

#### DELETE `/admin/accounts/{id}`
Soft-delete (deactivate) an account.

#### DELETE `/admin/accounts/{id}/hard`
Permanently delete an account and related data.

#### POST `/admin/accounts/{id}/activate`
Reactivate a deactivated account.

#### POST `/admin/accounts/{id}/regenerate`
Generate a new API token for an account.

**Response:**
```json
{
  "success": true,
  "token": "new-base64-token"
}
```

#### PUT `/admin/accounts/{id}/username`
Change account username.

**Request:**
```json
{
  "username": "newname"
}
```

#### PUT `/admin/accounts/{id}/password`
Set/change account password.

**Request:**
```json
{
  "password": "newPassword123"
}
```

#### PUT `/admin/accounts/{id}/organization`
Link account to an organization.

**Request:**
```json
{
  "orgId": "org-uuid"
}
```

#### PUT `/admin/accounts/{id}/org-admin`
Set organization admin status.

**Request:**
```json
{
  "isOrgAdmin": true
}
```

#### DELETE `/admin/accounts/{id}/totp`
Reset TOTP for an account (admin override).

---

### Organization Management

#### GET `/admin/organizations`
List all organizations.

**Response:**
```json
{
  "organizations": [
    {
      "id": "uuid",
      "name": "My Organization",
      "createdAt": "2024-01-01T00:00:00Z",
      "appCount": 5,
      "hasPolicy": true
    }
  ]
}
```

#### POST `/admin/organizations`
Create a new organization.

**Request:**
```json
{
  "name": "New Organization"
}
```

#### PUT `/admin/organizations/{id}`
Update organization name.

**Request:**
```json
{
  "name": "Updated Name"
}
```

#### DELETE `/admin/organizations/{id}`
Delete an organization (must have no applications).

#### GET `/admin/organizations/{id}/policy`
Get organization's auth policy.

**Response:**
```json
{
  "policy": {
    "orgId": "uuid",
    "authType": "oidc",
    "oidcIssuerUrl": "https://accounts.google.com",
    "oidcClientId": "client-id",
    "oidcScopes": ["openid", "email", "profile"],
    "oidcAllowedDomains": ["company.com"]
  }
}
```

#### PUT `/admin/organizations/{id}/policy`
Set organization's auth policy.

**Request (Basic Auth):**
```json
{
  "authType": "basic",
  "basicUsername": "username",
  "basicPassword": "password"
}
```

**Request (OIDC):**
```json
{
  "authType": "oidc",
  "oidcIssuerUrl": "https://accounts.google.com",
  "oidcClientId": "client-id",
  "oidcClientSecret": "client-secret",
  "oidcScopes": ["openid", "email", "profile"],
  "oidcAllowedDomains": ["company.com"],
  "oidcRequiredClaims": {
    "hd": "company.com"
  }
}
```

---

### Application Management

#### GET `/admin/applications`
List all applications.

**Query Parameters:**
- `org` - Filter by organization ID

**Response:**
```json
{
  "applications": [
    {
      "id": "uuid",
      "orgId": "org-uuid",
      "orgName": "My Organization",
      "subdomain": "myapp",
      "name": "My Application",
      "authMode": "inherit",
      "authType": "",
      "createdAt": "2024-01-01T00:00:00Z",
      "hasPolicy": false,
      "isActive": true,
      "activeTunnelCount": 1,
      "stats": {
        "totalConnections": 100,
        "bytesSent": 1048576,
        "bytesReceived": 2097152
      }
    }
  ]
}
```

#### POST `/admin/applications`
Create a new application.

**Request:**
```json
{
  "orgId": "org-uuid",
  "subdomain": "myapp",
  "name": "My Application"
}
```

#### GET `/admin/applications/{id}`
Get application details.

#### PUT `/admin/applications/{id}`
Update application.

**Request:**
```json
{
  "name": "Updated Name",
  "subdomain": "newsubdomain",
  "authMode": "custom",
  "authType": "basic"
}
```

#### DELETE `/admin/applications/{id}`
Delete an application.

#### GET `/admin/applications/{id}/stats`
Get application tunnel statistics.

#### GET `/admin/applications/{id}/tunnels`
Get active tunnels for an application.

#### GET `/admin/applications/{id}/policy`
Get application's custom auth policy.

#### PUT `/admin/applications/{id}/policy`
Set application's custom auth policy.

---

### API Key Management

#### GET `/admin/api-keys`
List API keys.

**Query Parameters (one required):**
- `org` - Filter by organization ID
- `app` - Filter by application ID

**Response:**
```json
{
  "keys": [
    {
      "id": "uuid",
      "orgId": "org-uuid",
      "appId": null,
      "keyType": "account",
      "keyPrefix": "dlk_abc1",
      "description": "CI/CD Pipeline",
      "createdAt": "2024-01-01T00:00:00Z",
      "lastUsed": "2024-01-15T12:00:00Z",
      "expiresAt": null
    }
  ]
}
```

#### POST `/admin/api-keys`
Create a new API key.

**Request:**
```json
{
  "orgId": "org-uuid",
  "appId": "optional-app-uuid",
  "description": "CI/CD Pipeline",
  "expiresIn": 30
}
```

> `expiresIn` is in days. Omit for non-expiring keys.

**Response:**
```json
{
  "success": true,
  "key": {
    "id": "uuid",
    "keyPrefix": "dlk_abc1",
    "description": "CI/CD Pipeline"
  },
  "rawKey": "dlk_full-key-value"
}
```

> ⚠️ The `rawKey` is only returned once at creation time.

#### DELETE `/admin/api-keys/{id}`
Revoke an API key.

---

### Whitelist Management

#### GET `/admin/whitelist`
List global whitelist entries.

#### POST `/admin/whitelist`
Add to global whitelist.

**Request:**
```json
{
  "ipRange": "10.0.0.0/8",
  "description": "Internal network"
}
```

#### DELETE `/admin/whitelist/{id}`
Remove from global whitelist.

#### GET `/admin/org-whitelists`
List all organization whitelist entries.

#### GET `/admin/app-whitelists`
List all application whitelist entries.

---

### Tunnel Management

#### GET `/admin/tunnels`
List active tunnels.

**Response:**
```json
{
  "active": [
    {
      "subdomain": "myapp",
      "url": "https://myapp.tunnel.digit.zone",
      "createdAt": "2024-01-15T12:00:00Z"
    }
  ],
  "records": [
    {
      "id": "uuid",
      "accountId": "account-uuid",
      "subdomain": "myapp",
      "clientIp": "1.2.3.4",
      "createdAt": "2024-01-15T12:00:00Z",
      "bytesSent": 1024,
      "bytesReceived": 2048
    }
  ]
}
```

---

### Statistics

#### GET `/admin/stats`
Get server statistics.

**Response:**
```json
{
  "activeTunnels": 5,
  "totalAccounts": 100,
  "activeAccounts": 95,
  "whitelistEntries": 10,
  "totalTunnels": 5000,
  "totalBytesSent": 1073741824,
  "totalBytesReceived": 2147483648
}
```

---

### Audit Log

#### GET `/admin/audit`
Get audit events.

**Query Parameters:**
- `org` - Filter by organization ID
- `app` - Filter by application ID
- `limit` - Results per page (default: 50, max: 100)
- `offset` - Pagination offset

**Response:**
```json
{
  "events": [
    {
      "id": "uuid",
      "timestamp": "2024-01-15T12:00:00Z",
      "orgId": "org-uuid",
      "appId": "app-uuid",
      "authType": "api_key",
      "success": true,
      "failureReason": null,
      "sourceIp": "1.2.3.4",
      "userIdentity": "api_key:dlk_abc1",
      "keyId": "key-uuid"
    }
  ],
  "total": 1000,
  "limit": 50,
  "offset": 0
}
```

#### GET `/admin/audit/stats`
Get authentication statistics.

---

## Auth API Endpoints

#### POST `/auth/check-account`
Check if an account exists and get login requirements.

**Request:**
```json
{
  "username": "admin"
}
```

**Response:**
```json
{
  "exists": true,
  "accountType": "admin",
  "requiresTotp": true,
  "orgName": ""
}
```

#### POST `/auth/login`
Login with username/password.

**Request:**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response (success, no TOTP):**
```json
{
  "success": true,
  "token": "jwt-token",
  "accountType": "admin",
  "orgId": "",
  "orgName": "",
  "isOrgAdmin": false
}
```

**Response (TOTP required):**
```json
{
  "success": true,
  "pendingToken": "pending-jwt",
  "needsTotp": true,
  "accountType": "admin"
}
```

**Response (TOTP setup required):**
```json
{
  "success": true,
  "pendingToken": "pending-jwt",
  "needsSetup": true,
  "accountType": "admin"
}
```

#### GET `/auth/totp/setup?token={pendingToken}`
Get TOTP setup information.

**Response:**
```json
{
  "success": true,
  "secret": "JBSWY3DPEHPK3PXP",
  "url": "otpauth://totp/..."
}
```

#### POST `/auth/totp/setup`
Verify TOTP code and enable TOTP.

**Request:**
```json
{
  "pendingToken": "pending-jwt",
  "code": "123456"
}
```

**Response:**
```json
{
  "success": true,
  "token": "jwt-token",
  "accountType": "admin"
}
```

#### POST `/auth/totp/verify`
Verify TOTP code for existing TOTP users.

**Request:**
```json
{
  "pendingToken": "pending-jwt",
  "code": "123456"
}
```

**Response:**
```json
{
  "success": true,
  "token": "jwt-token",
  "accountType": "admin"
}
```

---

## Org Portal API Endpoints

The Org Portal API mirrors many admin endpoints but scoped to the authenticated user's organization.

| Endpoint | Description |
|----------|-------------|
| GET `/org/me` | Current user's account |
| PUT `/org/me/password` | Change password |
| GET `/org/stats` | Organization statistics |
| GET `/org/accounts` | List org accounts |
| POST `/org/accounts` | Create org account |
| GET `/org/applications` | List org applications |
| POST `/org/applications` | Create application |
| GET `/org/whitelist` | List org whitelist |
| POST `/org/whitelist` | Add to whitelist |
| GET `/org/api-keys` | List API keys |
| POST `/org/api-keys` | Create API key |

---

## Tunnel Auth Endpoints (Per-Subdomain)

These endpoints are available on each application's subdomain (e.g., `https://myapp.tunnel.digit.zone/__auth/*`).

#### GET `/__auth/login?redirect={url}`
Start OIDC login flow.

#### GET `/__auth/callback`
OIDC callback (redirect from provider).

#### GET `/__auth/logout?redirect={url}`
Clear session and redirect.

#### GET `/__auth/health`
Auth health check for the subdomain.

**Response:**
```json
{
  "status": "ok",
  "subdomain": "myapp",
  "appId": "uuid",
  "authMode": "inherit",
  "hasOrgPolicy": true
}
```

---

## Public Endpoints

#### GET `/health`
Server health check.

**Response:**
```json
{
  "status": "ok",
  "activeTunnels": 5,
  "activeAccounts": 95,
  "whitelistEntries": 10
}
```

#### WebSocket `/_tunnel`
Tunnel client WebSocket endpoint.

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error description"
}
```

Common HTTP status codes:
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (missing/invalid auth)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found
- `409` - Conflict (e.g., duplicate username)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error

---

## Rate Limiting

Authentication endpoints are rate-limited:

- **Window**: 15 minutes
- **Max Attempts**: 10 per IP per window
- **Block Duration**: 30 minutes

Rate-limited responses include:
```
HTTP 429 Too Many Requests
Retry-After: 1800
```
