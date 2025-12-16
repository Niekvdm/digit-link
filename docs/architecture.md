# digit-link Architecture

## Overview

digit-link is a lightweight, production-ready tunnel system that exposes local services to the internet with built-in authentication, multi-tenancy, and comprehensive security controls. It follows a WebSocket-based architecture for real-time bidirectional communication between tunnel clients and the central server.

## System Design Philosophy

### Core Principles

1. **Security First**: Multiple authentication layers, fail-closed behavior, audit logging
2. **Multi-Tenancy**: Organization and application isolation with inherited policies
3. **Simplicity**: SQLite backend for easy deployment, minimal dependencies
4. **Observability**: Comprehensive audit logging and metrics

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                 INTERNET                                         │
└───────────────────────────────────┬─────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │       INGRESS (k8s/nginx)      │
                    │   - TLS Termination            │
                    │   - Wildcard DNS (*.tunnel.*)  │
                    └───────────────┬───────────────┘
                                    │
┌───────────────────────────────────┴───────────────────────────────────────────┐
│                          digit-link SERVER                                     │
│                                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                        HTTP ROUTER (server.go)                           │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐  │  │
│  │  │ /health      │  │ /_tunnel     │  │ /admin/*     │  │ /org/*      │  │  │
│  │  │ (public)     │  │ (WebSocket)  │  │ (admin API)  │  │ (org API)   │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘  │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │  │
│  │  │ /auth/*      │  │ /setup/*     │  │ /__auth/*    │                   │  │
│  │  │ (dashboard)  │  │ (first boot) │  │ (tunnel SSO) │                   │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                   │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                    │                                           │
│  ┌─────────────────────────────────┴─────────────────────────────────────────┐│
│  │                         CORE SERVICES                                      ││
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐                  ││
│  │  │ Auth          │  │ Policy        │  │ Rate          │                  ││
│  │  │ Middleware    │  │ Resolver      │  │ Limiter       │                  ││
│  │  └───────────────┘  └───────────────┘  └───────────────┘                  ││
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐                  ││
│  │  │ JWT Handler   │  │ OIDC Handler  │  │ Audit Logger  │                  ││
│  │  └───────────────┘  └───────────────┘  └───────────────┘                  ││
│  └───────────────────────────────────────────────────────────────────────────┘│
│                                    │                                           │
│  ┌─────────────────────────────────┴─────────────────────────────────────────┐│
│  │                         DATA LAYER (SQLite)                                ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        ││
│  │  │accounts  │ │orgs      │ │apps      │ │api_keys  │ │tunnels   │        ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘        ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐                     ││
│  │  │sessions  │ │policies  │ │audit_log │ │whitelist │                     ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘                     ││
│  └───────────────────────────────────────────────────────────────────────────┘│
│                                    │                                           │
│  ┌─────────────────────────────────┴─────────────────────────────────────────┐│
│  │                       TUNNEL MANAGER                                       ││
│  │  ┌───────────────────────────────────────────────────────────────────┐    ││
│  │  │  map[subdomain]*Tunnel                                             │    ││
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │    ││
│  │  │  │ myapp       │  │ api-dev     │  │ staging     │  ...            │    ││
│  │  │  │ → WebSocket │  │ → WebSocket │  │ → WebSocket │                 │    ││
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │    ││
│  │  └───────────────────────────────────────────────────────────────────┘    ││
│  └───────────────────────────────────────────────────────────────────────────┘│
└───────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ WebSocket
                    ┌───────────────┴───────────────┐
                    │        TUNNEL CLIENTS          │
                    │                                │
                    │  ┌──────────┐  ┌──────────┐   │
                    │  │ Client A │  │ Client B │   │
                    │  │ :3000    │  │ :8080    │   │
                    │  └──────────┘  └──────────┘   │
                    └────────────────────────────────┘
```

## Component Details

### 1. Server Core (`internal/server/`)

The server is the central hub handling all traffic routing.

```go
type Server struct {
    domain         string              // Base domain (tunnel.digit.zone)
    scheme         string              // URL scheme (https)
    db             *db.DB              // Database connection
    tunnels        map[string]*Tunnel  // Active tunnel connections
    mu             sync.RWMutex        // Tunnel map lock
    upgrader       websocket.Upgrader  // WebSocket upgrader
    authMiddleware *AuthMiddleware     // Auth enforcement
    oidcHandler    *auth.OIDCAuthHandler // OIDC SSO
}
```

**Key responsibilities:**
- HTTP request routing
- WebSocket connection management
- Tunnel registration and lifecycle
- Request forwarding through tunnels

### 2. Authentication Layer (`internal/auth/`)

Multi-method authentication supporting:

| Method | Use Case | Storage |
|--------|----------|---------|
| **JWT** | Dashboard sessions | Stateless (signed) |
| **API Token** | Tunnel client auth | SHA-256 hash in DB |
| **API Key** | Tunnel-level auth | SHA-256 hash in DB |
| **Basic Auth** | Simple tunnel protection | bcrypt hash in DB |
| **OIDC** | Enterprise SSO for tunnels | Sessions in DB |
| **TOTP** | 2FA for dashboard | Encrypted in DB |

### 3. Policy System (`internal/policy/`)

Hierarchical policy resolution:

```
Organization Policy (default)
         │
         ├─── App A (inherit) → Uses org policy
         │
         ├─── App B (disabled) → No auth required
         │
         └─── App C (custom) → Uses app-specific policy
```

```go
type EffectivePolicy struct {
    Type   AuthType      // basic, api_key, oidc
    OrgID  string
    AppID  string
    Basic  *BasicConfig  // Username/password hashes
    OIDC   *OIDCConfig   // OIDC provider config
}
```

### 4. Database Layer (`internal/db/`)

SQLite-based persistence with the following key design choices:

- **Foreign keys enabled** for referential integrity
- **Automatic migrations** for schema evolution
- **Soft deletes** for accounts (deactivation)
- **Indexed queries** for performance

### 5. Client (`internal/client/`)

The tunnel client is a CLI application with:

- WebSocket-based connection to server
- Automatic reconnection with exponential backoff
- Local HTTP proxy forwarding
- Interactive TUI for monitoring

## Request Flow

### Tunnel Client Registration

```
Client                     Server                      Database
   │                          │                            │
   │──[WebSocket Connect]────▶│                            │
   │                          │                            │
   │──[RegisterRequest]──────▶│                            │
   │   {subdomain, token}     │──[Validate Token]─────────▶│
   │                          │◀─[Account]─────────────────│
   │                          │                            │
   │                          │──[Check IP Whitelist]─────▶│
   │                          │◀─[Allowed/Denied]──────────│
   │                          │                            │
   │◀─[RegisterResponse]──────│──[Record Tunnel]──────────▶│
   │   {success, url}         │                            │
   │                          │                            │
   │◀─[Ping]──────────────────│ (every 30s)               │
   │──[Pong]─────────────────▶│                            │
```

### Public Request Through Tunnel

```
User                Server              AuthMiddleware        Tunnel Client
  │                   │                       │                    │
  │──[GET /api/data]─▶│                       │                    │
  │   Host: app.tun.* │                       │                    │
  │                   │──[AuthenticateRequest]│                    │
  │                   │◀─[AuthResult]─────────│                    │
  │                   │                       │                    │
  │                   │──[HTTPRequest]────────────────────────────▶│
  │                   │   {id, method, path, headers, body}        │
  │                   │                       │                    │
  │                   │◀─[HTTPResponse]───────────────────────────│
  │                   │   {id, status, headers, body}             │
  │                   │                       │                    │
  │◀─[200 OK]─────────│                       │                    │
```

## Multi-Tenancy Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     SYSTEM ADMIN (is_admin=true)                 │
│  - Full access to all organizations                              │
│  - Can create/manage organizations                               │
│  - Requires TOTP                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  Organization A │ │  Organization B │ │  Organization C │
│  ┌───────────┐  │ │  ┌───────────┐  │ │  ┌───────────┐  │
│  │ Org Admin │  │ │  │ Org Admin │  │ │  │ Org Admin │  │
│  └───────────┘  │ │  └───────────┘  │ │  └───────────┘  │
│  ┌───────────┐  │ │                 │ │                 │
│  │ Org User  │  │ │                 │ │                 │
│  └───────────┘  │ │                 │ │                 │
│                 │ │                 │ │                 │
│  Applications:  │ │  Applications:  │ │  Applications:  │
│  - app1.*       │ │  - service.*    │ │  - demo.*       │
│  - app2.*       │ │                 │ │                 │
│                 │ │                 │ │                 │
│  Whitelist:     │ │  Whitelist:     │ │  Whitelist:     │
│  - 10.0.0.0/8   │ │  - 192.168.1.0  │ │  - 0.0.0.0/0    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Backend** | Go 1.24 | Performance, concurrency, single binary |
| **Database** | SQLite | Simple deployment, sufficient for most use cases |
| **Frontend** | Vue 3 + TypeScript | Modern, reactive, type-safe |
| **WebSocket** | gorilla/websocket | Mature, well-tested library |
| **JWT** | golang-jwt/jwt | Standard JWT implementation |
| **OIDC** | coreos/go-oidc | Production-ready OIDC client |
| **Password** | bcrypt | Industry-standard password hashing |
| **TOTP** | pquerna/otp | RFC-compliant TOTP implementation |

## Deployment Architecture

### Kubernetes/k3s Deployment

```yaml
┌─────────────────────────────────────────────────────────────┐
│                        KUBERNETES CLUSTER                    │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    INGRESS CONTROLLER                  │  │
│  │  - Wildcard TLS: *.tunnel.digit.zone                  │  │
│  │  - TLS termination                                     │  │
│  │  - externalTrafficPolicy: Local (for real client IP)  │  │
│  └───────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              digit-link DEPLOYMENT                     │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │  Pod                                             │  │  │
│  │  │  - digit-link-server                             │  │  │
│  │  │  - Volume: /data (SQLite)                        │  │  │
│  │  │  - Env: DOMAIN, JWT_SECRET, TRUSTED_PROXIES      │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   PERSISTENT VOLUME                    │  │
│  │  - SQLite database                                     │  │
│  │  - Survives pod restarts                               │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server listen port | 8080 |
| `DOMAIN` | Base domain for tunnels | tunnel.digit.zone |
| `SCHEME` | URL scheme | https |
| `DB_PATH` | SQLite database path | data/digit-link.db |
| `JWT_SECRET` | JWT signing secret | Auto-generated (⚠️) |
| `TRUSTED_PROXIES` | Proxy IPs for X-Forwarded-For | (none) |
| `ADMIN_TOKEN` | Auto-create admin on startup | (none) |

## Design Decisions

### Why SQLite?

1. **Simplicity**: No external database to manage
2. **Performance**: Sufficient for thousands of concurrent tunnels
3. **Persistence**: Survives restarts with proper volume mounting
4. **ACID**: Full transaction support for data integrity

### Why WebSocket for Tunnels?

1. **Bidirectional**: Server can push requests to clients
2. **Persistent**: Long-lived connections reduce latency
3. **Firewall-Friendly**: Works through NAT/firewalls
4. **Efficient**: Lower overhead than HTTP polling

### Why Hierarchical Policies?

1. **DRY**: Set once at org level, inherit everywhere
2. **Flexibility**: Override per-application when needed
3. **Security**: Centralized policy management

## Future Considerations

1. **PostgreSQL Support**: For high-availability deployments
2. **Redis Session Store**: For horizontal scaling
3. **TCP Tunnel Support**: Beyond HTTP traffic
4. **Metrics Endpoint**: Prometheus-compatible metrics
5. **Webhook Notifications**: For tunnel events
