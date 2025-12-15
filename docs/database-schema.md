# Database Schema - Tunnel Authentication

This document describes the database tables used for tunnel-level authentication.

## Tables

### organizations

Top-level entity for grouping accounts and applications.

```sql
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### applications

Persistent applications with reserved subdomains.

```sql
CREATE TABLE applications (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id),
    subdomain TEXT UNIQUE NOT NULL,
    name TEXT,
    auth_mode TEXT DEFAULT 'inherit',  -- inherit, disabled, custom
    auth_type TEXT,                     -- basic, api_key, oidc (when custom)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Auth Modes:**
- `inherit` - Use organization's auth policy
- `disabled` - No authentication required
- `custom` - Use application's own auth policy

### org_auth_policies

Default authentication policy for an organization.

```sql
CREATE TABLE org_auth_policies (
    org_id TEXT PRIMARY KEY REFERENCES organizations(id),
    auth_type TEXT NOT NULL,           -- basic, api_key, oidc
    basic_user_hash TEXT,              -- bcrypt hash of username (optional)
    basic_pass_hash TEXT,              -- bcrypt hash of password
    oidc_issuer_url TEXT,              -- OIDC provider URL
    oidc_client_id TEXT,
    oidc_client_secret_enc TEXT,       -- Encrypted client secret
    oidc_scopes TEXT,                  -- JSON array of scopes
    oidc_allowed_domains TEXT,         -- JSON array of allowed email domains
    oidc_required_claims TEXT          -- JSON object of required claims
);
```

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

### api_keys

API keys for authentication.

```sql
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    org_id TEXT REFERENCES organizations(id),
    app_id TEXT REFERENCES applications(id),
    key_hash TEXT NOT NULL,            -- SHA-256 hash of the key
    key_prefix TEXT NOT NULL,          -- First 8 chars for identification
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    expires_at TIMESTAMP               -- NULL means never expires
);
```

**Key Scoping:**
- `org_id` set, `app_id` NULL → Key works for all apps in org
- `org_id` set, `app_id` set → Key only works for specific app

### auth_sessions

OIDC session storage.

```sql
CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,               -- 64-char hex session ID
    app_id TEXT,
    org_id TEXT,
    user_email TEXT,
    user_claims TEXT,                  -- JSON object of claims
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);
```

This table also stores OIDC state (with `id` prefix `oidc_state:`).

### rate_limit_state

Rate limiting state for auth endpoints.

```sql
CREATE TABLE rate_limit_state (
    key TEXT PRIMARY KEY,              -- e.g., "ip:1.2.3.4" or "app_ip:appid:1.2.3.4"
    count INTEGER DEFAULT 0,           -- Attempts in current window
    window_start TIMESTAMP,            -- Start of current window
    blocked_until TIMESTAMP            -- NULL if not blocked
);
```

### auth_audit_log

Audit log for authentication events.

```sql
CREATE TABLE auth_audit_log (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    org_id TEXT,
    app_id TEXT,
    auth_type TEXT,                    -- basic, api_key, oidc
    success BOOLEAN,
    failure_reason TEXT,               -- NULL on success
    source_ip TEXT,
    user_identity TEXT,                -- Username, email, or "api_key:prefix"
    key_id TEXT                        -- API key ID if applicable
);
```

## Indexes

```sql
-- Accounts (existing table, new column)
CREATE INDEX idx_accounts_org_id ON accounts(org_id);

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

-- Tunnels (existing table, new column)
CREATE INDEX idx_tunnels_app_id ON tunnels(app_id);
```

## Migrations

The schema includes automatic migrations for existing databases:

1. Add `org_id` column to `accounts` table
2. Add `app_id` column to `tunnels` table

New tables are created if they don't exist on startup.

## Data Relationships

```
organizations
    │
    ├── accounts (org_id)
    │
    ├── applications (org_id)
    │       │
    │       └── app_auth_policies (app_id)
    │
    ├── org_auth_policies (org_id)
    │
    └── api_keys (org_id, app_id)

auth_sessions ── linked to org_id and/or app_id
auth_audit_log ── references org_id and app_id
rate_limit_state ── standalone, keyed by IP/app
```
