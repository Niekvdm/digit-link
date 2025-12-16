# digit-link Database Schema

## Overview

digit-link uses SQLite for persistence. The database is stored at the path specified by `DB_PATH` environment variable (default: `data/digit-link.db`).

Foreign keys are enabled by default: `?_foreign_keys=on`

## Entity Relationship Diagram

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                ORGANIZATIONS                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │ id (PK) │ name (UNIQUE) │ require_totp │ created_at                     │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────┘
         │                    │                           │
         │ 1:N                │ 1:1                       │ 1:N
         ▼                    ▼                           ▼
┌─────────────────┐  ┌─────────────────────┐  ┌─────────────────────────────────┐
│    ACCOUNTS     │  │ ORG_AUTH_POLICIES   │  │          APPLICATIONS            │
│ ┌─────────────┐ │  │ ┌─────────────────┐ │  │ ┌─────────────────────────────┐ │
│ │ id (PK)     │ │  │ │ org_id (PK,FK)  │ │  │ │ id (PK)                     │ │
│ │ username    │ │  │ │ auth_type       │ │  │ │ org_id (FK)                 │ │
│ │ token_hash  │ │  │ │ basic_*         │ │  │ │ subdomain (UNIQUE)          │ │
│ │ password_*  │ │  │ │ oidc_*          │ │  │ │ name                        │ │
│ │ totp_*      │ │  │ └─────────────────┘ │  │ │ auth_mode                   │ │
│ │ is_admin    │ │  └─────────────────────┘  │ │ auth_type                   │ │
│ │ is_org_admin│ │                           │ └─────────────────────────────┘ │
│ │ org_id (FK) │ │                           └─────────────────────────────────┘
│ │ active      │ │                                    │
│ └─────────────┘ │                                    │ 1:1
└─────────────────┘                                    ▼
         │                                   ┌─────────────────────┐
         │                                   │ APP_AUTH_POLICIES   │
         │                                   │ ┌─────────────────┐ │
         │                                   │ │ app_id (PK,FK)  │ │
         │                                   │ │ auth_type       │ │
         │                                   │ │ basic_*         │ │
         │                                   │ │ oidc_*          │ │
         │                                   │ └─────────────────┘ │
         │                                   └─────────────────────┘
         │
         │ 1:N
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                           TUNNELS                                │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ id (PK) │ account_id (FK) │ subdomain │ client_ip │ app_id  │ │
│ │ created_at │ closed_at │ bytes_sent │ bytes_received        │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘

┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐
│       API_KEYS        │  │    AUTH_SESSIONS      │  │   AUTH_AUDIT_LOG      │
│ ┌───────────────────┐ │  │ ┌───────────────────┐ │  │ ┌───────────────────┐ │
│ │ id (PK)           │ │  │ │ id (PK)           │ │  │ │ id (PK)           │ │
│ │ org_id (FK)       │ │  │ │ app_id            │ │  │ │ timestamp         │ │
│ │ app_id (FK)       │ │  │ │ org_id            │ │  │ │ org_id            │ │
│ │ key_type          │ │  │ │ user_email        │ │  │ │ app_id            │ │
│ │ key_hash          │ │  │ │ user_claims       │ │  │ │ auth_type         │ │
│ │ key_prefix        │ │  │ │ expires_at        │ │  │ │ success           │ │
│ │ expires_at        │ │  │ └───────────────────┘ │  │ │ source_ip         │ │
│ └───────────────────┘ │  └───────────────────────┘  │ └───────────────────┘ │
└───────────────────────┘                             └───────────────────────┘

┌───────────────────────┐  ┌───────────────────────┐  ┌───────────────────────┐
│   GLOBAL_WHITELIST    │  │    ORG_WHITELIST      │  │    APP_WHITELIST      │
│ ┌───────────────────┐ │  │ ┌───────────────────┐ │  │ ┌───────────────────┐ │
│ │ id (PK)           │ │  │ │ id (PK)           │ │  │ │ id (PK)           │ │
│ │ ip_range          │ │  │ │ org_id (FK)       │ │  │ │ app_id (FK)       │ │
│ │ description       │ │  │ │ ip_range          │ │  │ │ ip_range          │ │
│ │ created_by (FK)   │ │  │ │ description       │ │  │ │ description       │ │
│ └───────────────────┘ │  │ └───────────────────┘ │  │ └───────────────────┘ │
└───────────────────────┘  └───────────────────────┘  └───────────────────────┘
```

## Table Definitions

### organizations

Top-level entity grouping accounts and applications.

```sql
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    require_totp BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `name` | TEXT | Unique organization name |
| `require_totp` | BOOLEAN | Force TOTP for all org members |
| `created_at` | TIMESTAMP | Creation timestamp |

---

### accounts

User accounts for dashboard access and tunnel client authentication.

```sql
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    token_hash TEXT NOT NULL,
    password_hash TEXT,
    totp_secret TEXT,
    totp_enabled BOOLEAN DEFAULT FALSE,
    is_admin BOOLEAN DEFAULT FALSE,
    is_org_admin BOOLEAN DEFAULT FALSE,
    org_id TEXT REFERENCES organizations(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    active BOOLEAN DEFAULT TRUE
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `username` | TEXT | Unique username |
| `token_hash` | TEXT | SHA-256 hash of API token |
| `password_hash` | TEXT | bcrypt hash of password (nullable) |
| `totp_secret` | TEXT | Encrypted TOTP secret (nullable) |
| `totp_enabled` | BOOLEAN | Whether TOTP is enabled |
| `is_admin` | BOOLEAN | System administrator flag |
| `is_org_admin` | BOOLEAN | Organization administrator flag |
| `org_id` | TEXT | FK to organization (nullable for admins) |
| `created_at` | TIMESTAMP | Account creation time |
| `last_used` | TIMESTAMP | Last token/login usage |
| `active` | BOOLEAN | Soft-delete flag |

**Security Notes:**
- `token_hash`: SHA-256 of the raw token (tokens are URL-safe base64)
- `password_hash`: bcrypt with cost factor 12
- `totp_secret`: AES-256-GCM encrypted

---

### applications

Persistent applications with reserved subdomains.

```sql
CREATE TABLE applications (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id),
    subdomain TEXT UNIQUE NOT NULL,
    name TEXT,
    auth_mode TEXT DEFAULT 'inherit',
    auth_type TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `org_id` | TEXT | FK to owning organization |
| `subdomain` | TEXT | Reserved subdomain (unique) |
| `name` | TEXT | Display name |
| `auth_mode` | TEXT | `inherit`, `disabled`, or `custom` |
| `auth_type` | TEXT | Auth type when mode=custom |
| `created_at` | TIMESTAMP | Creation timestamp |

**Auth Mode Values:**
- `inherit` - Use organization's default auth policy
- `disabled` - No authentication required
- `custom` - Use application's own auth policy

---

### org_auth_policies

Default authentication policy for an organization.

```sql
CREATE TABLE org_auth_policies (
    org_id TEXT PRIMARY KEY REFERENCES organizations(id),
    auth_type TEXT NOT NULL,
    basic_user_hash TEXT,
    basic_pass_hash TEXT,
    oidc_issuer_url TEXT,
    oidc_client_id TEXT,
    oidc_client_secret_enc TEXT,
    oidc_scopes TEXT,
    oidc_allowed_domains TEXT,
    oidc_required_claims TEXT
);
```

| Column | Type | Description |
|--------|------|-------------|
| `org_id` | TEXT | PK/FK to organization |
| `auth_type` | TEXT | `basic`, `api_key`, or `oidc` |
| `basic_user_hash` | TEXT | bcrypt hash of basic auth username |
| `basic_pass_hash` | TEXT | bcrypt hash of basic auth password |
| `oidc_issuer_url` | TEXT | OIDC provider URL |
| `oidc_client_id` | TEXT | OAuth2 client ID |
| `oidc_client_secret_enc` | TEXT | Client secret (⚠️ not encrypted) |
| `oidc_scopes` | TEXT | JSON array of scopes |
| `oidc_allowed_domains` | TEXT | JSON array of allowed email domains |
| `oidc_required_claims` | TEXT | JSON object of required claims |

---

### app_auth_policies

Custom authentication policy for an application (when `auth_mode = 'custom'`).

```sql
CREATE TABLE app_auth_policies (
    app_id TEXT PRIMARY KEY REFERENCES applications(id),
    auth_type TEXT NOT NULL,
    basic_user_hash TEXT,
    basic_pass_hash TEXT,
    oidc_issuer_url TEXT,
    oidc_client_id TEXT,
    oidc_client_secret_enc TEXT,
    oidc_scopes TEXT,
    oidc_allowed_domains TEXT,
    oidc_required_claims TEXT
);
```

Same structure as `org_auth_policies` but keyed by `app_id`.

---

### api_keys

API keys for tunnel-level and programmatic authentication.

```sql
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    org_id TEXT REFERENCES organizations(id),
    app_id TEXT REFERENCES applications(id),
    key_type TEXT DEFAULT 'account',
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    expires_at TIMESTAMP
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `org_id` | TEXT | FK to organization |
| `app_id` | TEXT | FK to application (nullable) |
| `key_type` | TEXT | `account` or `app` |
| `key_hash` | TEXT | SHA-256 hash of the key |
| `key_prefix` | TEXT | First 8 chars for identification |
| `description` | TEXT | User-provided description |
| `created_at` | TIMESTAMP | Creation timestamp |
| `last_used` | TIMESTAMP | Last usage timestamp |
| `expires_at` | TIMESTAMP | Expiration (nullable = never) |

**Key Scoping:**
- `org_id` set, `app_id` NULL → Works for all apps in organization
- `org_id` set, `app_id` set → Only works for specific application

**Key Types:**
- `account` - For random subdomain access (org-level)
- `app` - For specific application access only

---

### tunnels

Historical and active tunnel connection records.

```sql
CREATE TABLE tunnels (
    id TEXT PRIMARY KEY,
    account_id TEXT,
    subdomain TEXT NOT NULL,
    client_ip TEXT,
    app_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    bytes_sent INTEGER DEFAULT 0,
    bytes_received INTEGER DEFAULT 0,
    FOREIGN KEY(account_id) REFERENCES accounts(id)
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `account_id` | TEXT | FK to account (nullable for API keys) |
| `subdomain` | TEXT | Tunnel subdomain |
| `client_ip` | TEXT | Client's IP address |
| `app_id` | TEXT | FK to application (if known) |
| `created_at` | TIMESTAMP | Connection start time |
| `closed_at` | TIMESTAMP | Connection end time (nullable if active) |
| `bytes_sent` | INTEGER | Bytes sent through tunnel |
| `bytes_received` | INTEGER | Bytes received through tunnel |

---

### auth_sessions

OIDC session storage for tunnel-level authentication.

```sql
CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    app_id TEXT,
    org_id TEXT,
    user_email TEXT,
    user_claims TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | 64-char hex session ID |
| `app_id` | TEXT | Associated application |
| `org_id` | TEXT | Associated organization |
| `user_email` | TEXT | Authenticated user's email |
| `user_claims` | TEXT | JSON object of ID token claims |
| `created_at` | TIMESTAMP | Session creation time |
| `expires_at` | TIMESTAMP | Session expiration |

**Note:** This table also stores OIDC state entries (for CSRF protection) with `id` prefix `oidc_state:`.

---

### rate_limit_state

Rate limiting state for authentication endpoints.

```sql
CREATE TABLE rate_limit_state (
    key TEXT PRIMARY KEY,
    count INTEGER DEFAULT 0,
    window_start TIMESTAMP,
    blocked_until TIMESTAMP
);
```

| Column | Type | Description |
|--------|------|-------------|
| `key` | TEXT | Rate limit key (e.g., `ip:1.2.3.4`) |
| `count` | INTEGER | Attempts in current window |
| `window_start` | TIMESTAMP | Start of current window |
| `blocked_until` | TIMESTAMP | Block expiration (nullable) |

**Key Formats:**
- `ip:{ip}` - Per-IP rate limiting
- `app_ip:{app_id}:{ip}` - Per-app per-IP rate limiting
- `org_ip:{org_id}:{ip}` - Per-org per-IP rate limiting
- `user:{identity}` - Per-user rate limiting

---

### auth_audit_log

Audit log for all authentication events.

```sql
CREATE TABLE auth_audit_log (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    org_id TEXT,
    app_id TEXT,
    auth_type TEXT,
    success BOOLEAN,
    failure_reason TEXT,
    source_ip TEXT,
    user_identity TEXT,
    key_id TEXT
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `timestamp` | TIMESTAMP | Event timestamp |
| `org_id` | TEXT | Associated organization |
| `app_id` | TEXT | Associated application |
| `auth_type` | TEXT | `basic`, `api_key`, `oidc` |
| `success` | BOOLEAN | Whether auth succeeded |
| `failure_reason` | TEXT | Reason for failure (nullable) |
| `source_ip` | TEXT | Client IP address |
| `user_identity` | TEXT | Username, email, or key prefix |
| `key_id` | TEXT | API key ID if applicable |

---

### Whitelist Tables

Three-tier IP whitelisting system.

```sql
-- Legacy global whitelist
CREATE TABLE global_whitelist (
    id TEXT PRIMARY KEY,
    ip_range TEXT NOT NULL,
    description TEXT,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(created_by) REFERENCES accounts(id)
);

-- Organization-level whitelist
CREATE TABLE org_whitelist (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    ip_range TEXT NOT NULL,
    description TEXT,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(created_by) REFERENCES accounts(id)
);

-- Application-level whitelist
CREATE TABLE app_whitelist (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    ip_range TEXT NOT NULL,
    description TEXT,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(created_by) REFERENCES accounts(id)
);

-- Account-level whitelist (for tunnel client connections)
CREATE TABLE account_whitelist (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    ip_range TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE
);
```

**IP Range Format:**
- Single IP: `192.168.1.1`
- CIDR range: `10.0.0.0/8`

---

## Indexes

```sql
-- Accounts
CREATE INDEX idx_accounts_username ON accounts(username);
CREATE INDEX idx_accounts_token_hash ON accounts(token_hash);
CREATE INDEX idx_accounts_org_id ON accounts(org_id);

-- Tunnels
CREATE INDEX idx_tunnels_account_id ON tunnels(account_id);
CREATE INDEX idx_tunnels_subdomain ON tunnels(subdomain);
CREATE INDEX idx_tunnels_app_id ON tunnels(app_id);

-- Whitelists
CREATE INDEX idx_global_whitelist_ip ON global_whitelist(ip_range);
CREATE INDEX idx_org_whitelist_org_id ON org_whitelist(org_id);
CREATE INDEX idx_org_whitelist_ip ON org_whitelist(ip_range);
CREATE INDEX idx_app_whitelist_app_id ON app_whitelist(app_id);
CREATE INDEX idx_app_whitelist_ip ON app_whitelist(ip_range);

-- Applications
CREATE INDEX idx_applications_subdomain ON applications(subdomain);
CREATE INDEX idx_applications_org_id ON applications(org_id);

-- API Keys
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_org_id ON api_keys(org_id);
CREATE INDEX idx_api_keys_app_id ON api_keys(app_id);

-- Sessions
CREATE INDEX idx_auth_sessions_expires ON auth_sessions(expires_at);

-- Audit Log
CREATE INDEX idx_auth_audit_log_timestamp ON auth_audit_log(timestamp);
CREATE INDEX idx_auth_audit_log_org_id ON auth_audit_log(org_id);
CREATE INDEX idx_auth_audit_log_app_id ON auth_audit_log(app_id);
```

---

## Migrations

The database schema includes automatic migrations for existing databases:

```go
columnMigrations := []struct {
    table  string
    column string
    def    string
}{
    {"accounts", "password_hash", "TEXT"},
    {"accounts", "totp_secret", "TEXT"},
    {"accounts", "totp_enabled", "BOOLEAN DEFAULT FALSE"},
    {"accounts", "org_id", "TEXT REFERENCES organizations(id)"},
    {"accounts", "is_org_admin", "BOOLEAN DEFAULT FALSE"},
    {"tunnels", "app_id", "TEXT"},
    {"api_keys", "key_type", "TEXT DEFAULT 'account'"},
    {"organizations", "require_totp", "BOOLEAN DEFAULT FALSE"},
}
```

Migrations run automatically on startup and are idempotent.

---

## Query Patterns

### Authentication Queries

```sql
-- Validate token (tunnel client auth)
SELECT * FROM accounts 
WHERE token_hash = ? AND active = TRUE;

-- Validate API key
SELECT * FROM api_keys 
WHERE key_hash = ? 
AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);

-- Validate session
SELECT * FROM auth_sessions 
WHERE id = ? 
AND expires_at > CURRENT_TIMESTAMP
AND (app_id IS NULL OR app_id = ?)
AND (org_id IS NULL OR org_id = ?);
```

### IP Whitelist Queries

```sql
-- Check if IP is whitelisted for account
SELECT COUNT(*) FROM account_whitelist 
WHERE account_id = ?;
-- If count > 0, then check if IP matches any range

-- Check if IP is whitelisted for org
SELECT COUNT(*) FROM org_whitelist 
WHERE org_id = ?;

-- Check if IP is whitelisted for app
SELECT COUNT(*) FROM app_whitelist 
WHERE app_id = ?;
```

### Statistics Queries

```sql
-- Tunnel stats by app
SELECT 
    COUNT(*) as total_connections,
    SUM(bytes_sent) as total_bytes_sent,
    SUM(bytes_received) as total_bytes_received
FROM tunnels 
WHERE app_id = ?;

-- Auth stats
SELECT 
    COUNT(*) as total,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failures
FROM auth_audit_log 
WHERE timestamp > ?;
```

---

## Backup and Recovery

### Backup

SQLite database can be backed up while running:

```bash
sqlite3 data/digit-link.db ".backup 'backup.db'"
```

Or using the online backup API in Go:
```go
_, err := db.conn.Exec(`VACUUM INTO ?`, backupPath)
```

### Recovery

1. Stop the server
2. Replace database file
3. Start the server

The server will run migrations automatically if needed.
