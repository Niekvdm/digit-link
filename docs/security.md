# digit-link Security Assessment

## Executive Summary

digit-link implements a defense-in-depth security model with multiple authentication layers, rate limiting, audit logging, and IP whitelisting. This document outlines the security architecture, risk assessments, and safeguards in place.

## Security Architecture

### Authentication Layers

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           SECURITY LAYERS                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  LAYER 1: Network (Tunnel Client Authentication)                        │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  • API Token (SHA-256 hashed)                                       │ │
│  │  • API Key (SHA-256 hashed)                                         │ │
│  │  • IP Whitelist enforcement                                         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  LAYER 2: Application (Tunnel Traffic Authentication)                   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  • Basic Auth (bcrypt hashed)                                       │ │
│  │  • API Key (SHA-256 hashed)                                         │ │
│  │  • OIDC/OAuth2 (PKCE flow)                                          │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  LAYER 3: Dashboard (Admin/Org Portal Authentication)                   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  • Password (bcrypt, cost 12)                                       │ │
│  │  • TOTP (RFC 6238, required for admins)                             │ │
│  │  • JWT Sessions (HS256, 24h expiry)                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Cryptographic Standards

### Password Hashing

| Algorithm | Cost | Library |
|-----------|------|---------|
| bcrypt | 12 | golang.org/x/crypto/bcrypt |

```go
const BcryptCost = 12

func HashPassword(password string) (string, error) {
    if len(password) < 8 {
        return "", fmt.Errorf("password must be at least 8 characters")
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
    return string(hash), err
}
```

### Token Hashing

| Algorithm | Purpose | Library |
|-----------|---------|---------|
| SHA-256 | API tokens, API keys | crypto/sha256 |

```go
func HashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
```

### Token Generation

| Size | Encoding | Library |
|------|----------|---------|
| 32 bytes | URL-safe Base64 | crypto/rand |

```go
const TokenLength = 32

func GenerateToken() (token string, hash string, err error) {
    bytes := make([]byte, TokenLength)
    if _, err := rand.Read(bytes); err != nil {
        return "", "", err
    }
    token = base64.URLEncoding.EncodeToString(bytes)
    hash = HashToken(token)
    return token, hash, nil
}
```

### JWT Configuration

| Algorithm | Secret | Expiration |
|-----------|--------|------------|
| HS256 | 32+ bytes (from JWT_SECRET env) | 24 hours |

```go
const JWTExpiration = 24 * time.Hour

type JWTClaims struct {
    AccountID string `json:"accountId"`
    Username  string `json:"username"`
    IsAdmin   bool   `json:"isAdmin"`
    OrgID     string `json:"orgId,omitempty"`
    jwt.RegisteredClaims
}
```

### TOTP Configuration

| Algorithm | Digits | Period | Window |
|-----------|--------|--------|--------|
| SHA1 | 6 | 30s | ±1 period |

**TOTP Secret Encryption:**
- Algorithm: AES-256-GCM
- Key derivation: From JWT_SECRET

### OIDC Security

| Feature | Implementation |
|---------|----------------|
| PKCE | S256 challenge |
| State | Cryptographic random |
| Nonce | Verified in ID token |
| Session | HttpOnly, Secure, SameSite=Lax |

---

## Risk Assessment

### Critical Risks

#### RISK-001: OIDC Client Secrets Stored Unencrypted
**Severity:** Critical  
**Location:** `org_auth_policies.oidc_client_secret_enc`, `app_auth_policies.oidc_client_secret_enc`  
**Status:** ⚠️ Open (TODO in code)

**Description:** OIDC client secrets are stored in plaintext in the database despite the column name suggesting encryption.

**Impact:** Database compromise exposes OIDC client secrets, allowing attackers to impersonate the application to the OIDC provider.

**Mitigation:**
1. Implement encryption using AES-256-GCM (same as TOTP secrets)
2. Use a separate encryption key from JWT_SECRET
3. Consider using a secrets manager (Vault, AWS Secrets Manager)

#### RISK-002: HSTS Header Bug
**Severity:** Critical  
**Location:** `internal/auth/headers.go:60`  

**Description:** HSTS header is generated incorrectly due to `string(rune(int))` conversion bug.

**Impact:** HSTS is not properly enforced, leaving users vulnerable to SSL stripping attacks.

**Mitigation:** Use `strconv.Itoa()` for integer-to-string conversion.

### High Risks

#### RISK-003: Missing Rate Limiting on Auth Endpoints
**Severity:** High  
**Location:** `internal/server/auth.go`

**Description:** While tunnel auth has rate limiting via middleware, the `/auth/login` endpoint for dashboard authentication doesn't use the rate limiter.

**Impact:** Brute-force attacks on dashboard credentials are not throttled.

**Mitigation:** Apply `RateLimiter.Allow()` check before processing login requests.

#### RISK-004: OIDC Provider Cache Race Condition
**Severity:** High  
**Location:** `internal/auth/oidc.go:505-511`

**Description:** The `UpdateProviderRedirectURL()` method mutates a shared cached provider's OAuth2 config.

**Impact:** Concurrent requests for different subdomains can cause OIDC callbacks to fail or redirect to wrong subdomain.

**Mitigation:** Create per-request OAuth2 config or use subdomain-keyed provider cache.

#### RISK-005: Auto-Generated JWT Secret
**Severity:** High  
**Location:** `internal/auth/jwt.go:36-46`

**Description:** If `JWT_SECRET` is not set, a random secret is generated. This breaks sessions across restarts.

**Impact:** All user sessions invalidated on server restart; potential for session confusion in replicated deployments.

**Mitigation:** 
1. Fail startup if `JWT_SECRET` not set in production
2. Use `log.Printf` instead of `fmt.Println` for warning
3. Document required environment variables

### Medium Risks

#### RISK-006: JWT in localStorage
**Severity:** Medium  
**Location:** `frontend/src/stores/authStore.ts:39`

**Description:** JWT tokens are stored in `localStorage`, making them accessible to JavaScript.

**Impact:** XSS vulnerabilities anywhere in the application could steal authentication tokens.

**Mitigation:**
1. Use HttpOnly cookies for session management
2. Implement robust CSP headers
3. Consider short-lived access tokens with refresh tokens

#### RISK-007: WebSocket Origin Not Validated
**Severity:** Medium  
**Location:** `internal/server/server.go:51-53`

**Description:** WebSocket `CheckOrigin` always returns `true`.

**Impact:** Malicious websites could establish WebSocket connections on behalf of users (CSRF-style attack).

**Mitigation:** Validate origin against expected domain:
```go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    return origin == "" || strings.HasSuffix(origin, domain)
}
```

#### RISK-008: Reserved Subdomains Not Blocked
**Severity:** Medium  
**Location:** `internal/server/server.go:711-722`

**Description:** Users can register subdomains like `admin`, `api`, `www`, etc.

**Impact:** Confusion attacks, potential phishing within the tunnel domain.

**Mitigation:** Maintain a blocklist of reserved subdomains:
```go
var reservedSubdomains = []string{"admin", "api", "www", "auth", "dashboard", ...}
```

### Low Risks

#### RISK-009: No Connection Timeout on Tunnel Messages
**Severity:** Low  
**Location:** `internal/server/server.go:586-613`

**Description:** No read deadline set during tunnel message handling.

**Impact:** Malicious clients could hold goroutines indefinitely.

**Mitigation:** Set and reset read deadlines based on ping/pong:
```go
conn.SetReadDeadline(time.Now().Add(pongWait))
```

#### RISK-010: Database Not Connection Pooled
**Severity:** Low  
**Location:** `internal/db/db.go:28`

**Description:** SQLite connection pool settings not configured.

**Impact:** Performance issues under high concurrency.

**Mitigation:** Configure pool settings:
```go
db.SetMaxOpenConns(1) // SQLite supports single writer
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
```

---

## Security Headers

### Default Headers

Applied to all responses:

```go
// Content-Security-Policy
"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none';"

// X-Frame-Options
"DENY"

// X-Content-Type-Options
"nosniff"

// X-XSS-Protection
"1; mode=block"

// Referrer-Policy
"strict-origin-when-cross-origin"

// Permissions-Policy
"geolocation=(), microphone=(), camera=()"
```

### Auth Endpoint Headers

Additional restrictions for auth endpoints:

```go
// More restrictive CSP
"default-src 'self'; script-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self';"

// No-cache headers
"Cache-Control: no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0"
"Pragma: no-cache"
"Expires: 0"
```

### HSTS (when fixed)

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

---

## Rate Limiting

### Configuration

| Setting | Value |
|---------|-------|
| Window Duration | 15 minutes |
| Max Attempts | 10 per window |
| Block Duration | 30 minutes |
| Cleanup Interval | 5 minutes |

### Implementation

- **Storage:** SQLite-backed with in-memory cache
- **Persistence:** Survives server restarts
- **Keys:** IP-based, with optional app/org scoping

```go
type RateLimiterConfig struct {
    WindowDuration  time.Duration  // 15 * time.Minute
    MaxAttempts     int            // 10
    BlockDuration   time.Duration  // 30 * time.Minute
    CleanupInterval time.Duration  // 5 * time.Minute
}
```

---

## Audit Logging

### Logged Events

| Event | Fields |
|-------|--------|
| Authentication Success | timestamp, org_id, app_id, auth_type, source_ip, user_identity, key_id |
| Authentication Failure | + failure_reason |
| Rate Limit Hit | failure_reason = "rate_limited" |

### Retention

Audit logs should be periodically cleaned:
```go
// Delete events older than 90 days
deleted, err := db.DeleteOldAuditEvents(90 * 24 * time.Hour)
```

---

## IP Whitelisting

### Hierarchy

1. **Global Whitelist** (legacy) - System-wide
2. **Organization Whitelist** - Per-org tunnel client access
3. **Application Whitelist** - Per-app tunnel client access
4. **Account Whitelist** - Per-account tunnel client access

### CIDR Support

Both single IPs and CIDR ranges are supported:
- `192.168.1.1`
- `10.0.0.0/8`
- `2001:db8::/32` (IPv6)

### Trusted Proxies

For proper client IP detection behind load balancers:

```bash
# Trust all private ranges (recommended for k8s)
TRUSTED_PROXIES=private

# Trust specific ranges
TRUSTED_PROXIES=10.42.0.0/16,172.16.0.0/12

# Trust all (not recommended)
TRUSTED_PROXIES=*
```

---

## Fail-Closed Behavior

The system defaults to denying access when errors occur:

| Scenario | Behavior |
|----------|----------|
| Policy load failure | Deny request |
| OIDC provider unreachable | Deny request |
| Session validation error | Redirect to login |
| Rate limit state error | Allow (degrade gracefully) |
| Database connection error | Deny request |

---

## Recommendations

### Immediate Actions (P1)

1. ✅ Fix HSTS header generation bug
2. ✅ Implement OIDC client secret encryption
3. ✅ Add rate limiting to `/auth/login` endpoint
4. ✅ Fix OIDC provider cache race condition

### Short-Term (P2)

5. Add reserved subdomain blocklist
6. Validate WebSocket origins
7. Add WebSocket read deadlines
8. Make JWT_SECRET required in production

### Medium-Term (P3)

9. Migrate JWT storage to HttpOnly cookies
10. Implement token rotation for long-lived sessions
11. Add anomaly detection for auth patterns
12. Implement security event webhooks

### Long-Term (P4)

13. Add support for hardware security keys (WebAuthn)
14. Implement session binding (device fingerprinting)
15. Add geographic rate limiting
16. Support external audit log shipping (SIEM integration)

---

## Compliance Considerations

### OWASP Top 10 Coverage

| Risk | Status | Implementation |
|------|--------|----------------|
| A01:2021 Broken Access Control | ✅ | Role-based access, policy system |
| A02:2021 Cryptographic Failures | ⚠️ | bcrypt, SHA-256 (OIDC secrets unencrypted) |
| A03:2021 Injection | ✅ | Parameterized SQL queries |
| A04:2021 Insecure Design | ✅ | Defense in depth |
| A05:2021 Security Misconfiguration | ⚠️ | Secure defaults (JWT secret issue) |
| A06:2021 Vulnerable Components | ✅ | Modern Go dependencies |
| A07:2021 Auth Failures | ⚠️ | Strong auth (rate limiting gap) |
| A08:2021 Data Integrity Failures | ✅ | JWT signature verification |
| A09:2021 Security Logging | ✅ | Comprehensive audit log |
| A10:2021 SSRF | ✅ | Limited outbound connections |

### GDPR Considerations

- Audit logs may contain PII (email, IP addresses)
- Implement data retention policies
- Support data export for subject access requests
- Document data processing in privacy policy
