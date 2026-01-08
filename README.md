# digit-link

A lightweight ngrok-like tunnel system for exposing local services to the internet, with built-in authentication, multi-tenancy, and enterprise SSO support.

## Features

- **Secure Tunneling**: TCP+TLS tunnels with yamux multiplexing for high performance
- **Multi-Forward**: Expose multiple local services through a single connection
- **Interactive TUI**: Real-time request monitoring with filtering and stats
- **Multi-Tenancy**: Organizations and Applications for resource isolation
- **Authentication Options**: Basic Auth, API Keys, or OIDC/SSO for tunnel access
- **IP Whitelisting**: Global, organization, application, and account-level controls
- **Admin Dashboard**: Modern Vue 3 web interface for management
- **Rate Limiting**: Protection against brute-force attacks
- **SQLite Storage**: Zero-dependency persistent storage
- **Static Binaries**: Client builds with no external dependencies

## Quick Start

### Server

```bash
# Build and run
make build-server
./build/bin/digit-link-server
```

On first boot, navigate to your domain in a browser. The **setup wizard** will guide you through creating your admin account.

Alternative CLI setup:
```bash
./build/bin/digit-link-server --setup-admin
```

### Client

```bash
# Build the client (static binary, no dependencies)
make build-client

# Interactive mode - launches setup TUI
./build/bin/digit-link

# Or with command-line flags
./build/bin/digit-link \
  --server link.digit.zone \
  --subdomain myapp \
  --port 3000 \
  --token YOUR_TOKEN

# Multi-forward: expose multiple services
./build/bin/digit-link \
  --server link.digit.zone \
  --token YOUR_TOKEN \
  --forward myapp:3000 \
  --forward api:8080 \
  --forward admin:9000

# Your services are available at:
# https://myapp.link.digit.zone
# https://api.link.digit.zone
# https://admin.link.digit.zone
```

### Client Options

| Flag | Description | Default |
|------|-------------|---------|
| `--server` | Tunnel server address | `link.digit.zone` |
| `--subdomain` | Subdomain to register | - |
| `--port` | Local port to forward | - |
| `--forward` | Forward definition (subdomain:port) | - |
| `--token` | Authentication token | - |
| `--local-https` | Forward to local HTTPS server | `false` |
| `--insecure` | Skip TLS verification | `false` |

### Interactive TUI

The client includes an interactive terminal UI with:
- Real-time request monitoring
- Connection status and uptime
- Request filtering (press `/`)
- Copy URL to clipboard (press `c`)
- Stats tabs (press `Tab`)

### Configuration

The client saves configuration to:
- **Windows**: `%APPDATA%\digit-link\config.json`
- **macOS**: `~/Library/Application Support/digit-link/config.json`
- **Linux**: `~/.config/digit-link/config.json`

## Environment Variables

### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DOMAIN` | Base domain for tunnels | `link.digit.zone` |
| `DB_PATH` | SQLite database path | `data/digit-link.db` |
| `JWT_SECRET` | Secret for JWT tokens | (auto-generated) |
| `ADMIN_TOKEN` | Auto-create admin on startup | (none) |
| `TRUSTED_PROXIES` | Trusted proxy IPs/CIDRs | (none) |

### Client

| Variable | Description |
|----------|-------------|
| `DIGIT_LINK_TOKEN` | Alternative to `--token` flag |

### Trusted Proxies

When running behind a reverse proxy or in Kubernetes:

```bash
# Trust all private IP ranges (recommended for k8s/Docker)
TRUSTED_PROXIES=private

# Trust specific CIDR
TRUSTED_PROXIES=10.42.0.0/16

# Trust specific IPs
TRUSTED_PROXIES=10.0.0.1,10.0.0.2
```

**Important**: Configure your ingress with `externalTrafficPolicy: Local` to preserve client IPs.

## Building

```bash
make build          # Build frontend + server + client
make build-client   # Build client only (static binary, no CGO)
make build-server   # Build server only (requires CGO for SQLite)
make build-all      # Cross-compile for all platforms
```

**Client binaries** are fully static (`CGO_ENABLED=0`) and require no dependencies on target machines.

**Server binaries** require CGO for SQLite support.

## Deployment

The server is designed for containerized deployment. See [Deployment Architecture](docs/architecture.md#deployment-architecture) for details.

### DNS Setup

```
*.tunnel    A    <INGRESS_IP>
tunnel      A    <INGRESS_IP>
```

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

| Document | Description |
|----------|-------------|
| [**Architecture**](docs/architecture.md) | System design, components, data flow, deployment |
| [**API Reference**](docs/api.md) | Complete REST API documentation with examples |
| [**Database**](docs/database.md) | Schema, relationships, ERD, query patterns |
| [**Security**](docs/security.md) | Security architecture, risk assessment, safeguards |
| [**Performance**](docs/performance.md) | Tuning, scaling, monitoring |
| [**Frontend**](docs/frontend.md) | Vue 3 architecture, components, state management |

### Quick Links

**For Developers**
- [API Endpoints](docs/api.md#admin-api-endpoints)
- [Database Schema](docs/database.md#table-definitions)
- [Frontend Components](docs/frontend.md#components)

**For DevOps**
- [Environment Variables](docs/architecture.md#environment-variables)
- [Performance Tuning](docs/performance.md#performance-tuning-checklist)
- [Deployment](docs/architecture.md#deployment-architecture)

**For Security Teams**
- [Security Architecture](docs/security.md#security-architecture)
- [Risk Assessment](docs/security.md#risk-assessment)
- [Authentication Flow](docs/security.md#authentication--authorization)

## How It Works

```
┌─────────────┐     TCP+TLS        ┌─────────────┐     HTTP        ┌──────────┐
│   Client    │◄──────────────────►│   Server    │◄───────────────►│ Internet │
│ (local:3000)│    yamux mux       │  (k3s)      │   subdomain     │          │
└─────────────┘                    └─────────────┘                 └──────────┘
```

1. Client connects via TCP+TLS with yamux multiplexing
2. Server validates token and IP whitelist
3. Server registers subdomain(s) and provides public URL(s)
4. Incoming HTTP requests are forwarded to client over multiplexed streams
5. Client forwards to local service and returns response
6. PROXY protocol preserves original client IPs

## Security Highlights

- **Token Security**: Generated with `crypto/rand`, stored as SHA-256 hashes
- **Password Hashing**: bcrypt with cost factor 12
- **Session Management**: JWT with configurable expiry
- **Rate Limiting**: Per-IP and per-application rate limiting
- **OIDC/SSO**: PKCE-secured OAuth2 flows for enterprise SSO
- **IP Controls**: Multi-tiered whitelisting (global → org → app → account)

See [Security Documentation](docs/security.md) for full details.

## License

MIT
