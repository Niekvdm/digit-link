# digit-link

A lightweight ngrok-like tunnel system for exposing local services to the internet, with built-in authentication and IP whitelisting.

## Features

- **Token-based Authentication**: Secure API tokens for client connections
- **Tunnel-Level Auth**: Protect public access to tunnels with Basic, API Key, or OIDC
- **IP Whitelisting**: Strict control over which IPs can connect
- **Admin Dashboard**: Web-based management interface
- **SQLite Storage**: Persistent account and tunnel data
- **Automatic Reconnection**: Clients auto-reconnect on disconnection

## Architecture

- **Server**: Runs on k3s, handles WebSocket connections from clients and routes HTTP traffic
- **Client**: CLI tool that connects to the server and forwards requests to local ports
- **Admin Dashboard**: Web UI for managing accounts, IP whitelist, and monitoring tunnels

## Documentation

- [Tunnel-Level Authentication](docs/tunnel-auth.md) - Protect public access to tunnels
- [Database Schema](docs/database-schema.md) - Complete database table reference
- [API Reference](docs/api-reference.md) - Programmatic API for auth management
- [OIDC Configuration](docs/oidc-configuration.md) - Configure OIDC providers

## Quick Start

### Server Setup

```bash
# Build the server
make build-server

# Run the server
./build/bin/digit-link-server
```

#### First-Boot Setup (Recommended)

When the server starts with no admin account, navigate to your domain in a browser. The **first-boot setup wizard** will guide you through creating your admin account and generating your token.

This is ideal for Kubernetes/k3s deployments where CLI access is limited.

#### Alternative: CLI Setup

```bash
# Create initial admin account via command line
./build/bin/digit-link-server --setup-admin
```

On first run with `--setup-admin`, you'll receive an admin token. Save it securely!

### Client Usage

```bash
# Build the client
make build-client

# Connect to tunnel server
./build/bin/digit-link --server tunnel.digit.zone --subdomain myapp --port 3000 --token YOUR_TOKEN

# Your local service is now available at:
# https://myapp.tunnel.digit.zone
```

### Options

```
Client:
--server     Tunnel server address (default: tunnel.digit.zone)
--subdomain  Subdomain to register (required)
--port       Local port to forward to (required)
--token      Authentication token (required)
--timeout    Request timeout for slow endpoints (default: 5m, e.g., 10m, 1h)

Server:
--setup-admin     Create initial admin account and exit
--admin-username  Username for initial admin account (default: admin)
```

## Admin Dashboard

Access the admin dashboard at `https://tunnel.digit.zone/` (or your server domain).

Features:
- **Accounts**: Create and manage user accounts, generate tokens
- **Whitelist**: Add/remove IP addresses or CIDR ranges
- **Tunnels**: Monitor active tunnel connections in real-time

## Environment Variables

### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DOMAIN` | Base domain for tunnels | `tunnel.digit.zone` |
| `SECRET` | Legacy shared secret (deprecated) | (none) |
| `DB_PATH` | SQLite database path | `data/digit-link.db` |
| `ADMIN_TOKEN` | Auto-create admin with this token on startup | (none) |
| `JWT_SECRET` | Secret for signing JWT tokens (auto-generated if not set) | (auto) |
| `TRUSTED_PROXIES` | Trusted proxy IPs/CIDRs for X-Forwarded-For (see below) | (none) |

#### TRUSTED_PROXIES Configuration

When running behind a reverse proxy, load balancer, or in Kubernetes, the server sees the proxy's IP instead of the client's real IP. Set `TRUSTED_PROXIES` to tell digit-link which proxies to trust for forwarded headers.

**Values:**
- `private` - Trust all private IP ranges (recommended for k8s/Docker)
- `10.42.0.0/16` - Trust specific CIDR range
- `10.0.0.1,10.0.0.2` - Trust specific IPs (comma-separated)
- `*` - Trust all IPs (not recommended for production)

**Kubernetes/k3s example:**
```yaml
env:
  - name: TRUSTED_PROXIES
    value: "private"
```

**Docker example:**
```bash
docker run -e TRUSTED_PROXIES=private digit-link-server
```

**Important:** Your ingress or reverse proxy must be configured to set `X-Forwarded-For` or `X-Real-IP` headers. For Traefik (k3s default), this is typically automatic.

### Client

| Variable | Description |
|----------|-------------|
| `DIGIT_LINK_TOKEN` | Authentication token (alternative to `--token` flag) |

## Building

```bash
# Build for current platform
make build

# Build for specific platforms
make build-windows
make build-linux
make build-darwin
make build-darwin-arm

# Build all platforms
make build-all
```

## Server Deployment

The server is deployed to k3s via ArgoCD. See the `k8s/` directory for manifests.

### DNS Setup

Point your wildcard DNS to the k3s ingress:

```
*.tunnel    A    <K3S_INGRESS_IP>
tunnel      A    <K3S_INGRESS_IP>
```

## Security

- **Tokens**: Generated using `crypto/rand`, stored as SHA-256 hashes
- **IP Whitelisting**: Only whitelisted IPs can establish tunnels
- **HTTPS**: All traffic should be served over HTTPS (via ingress)
- **Admin API**: Protected by admin token header (`X-Admin-Token`)
- **Tunnel Auth**: Optional authentication on public tunnel traffic (Basic/API Key/OIDC)
- **Rate Limiting**: Protection against brute-force attacks on auth endpoints
- **Audit Logging**: All authentication events are logged

## How It Works

1. Client connects to server via WebSocket at `wss://tunnel.digit.zone/_tunnel`
2. Client sends registration with subdomain and authentication token
3. Server validates token and checks if client IP is whitelisted
4. On success, server registers the tunnel and provides public URL
5. When requests arrive at the public URL, server forwards them to the client over WebSocket
6. Client forwards requests to the local service and returns responses

## API Endpoints

### Admin API (requires `X-Admin-Token` header)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/accounts` | GET | List all accounts |
| `/admin/accounts` | POST | Create new account |
| `/admin/accounts/{id}` | DELETE | Deactivate account |
| `/admin/accounts/{id}/regenerate` | POST | Regenerate token |
| `/admin/whitelist` | GET | List whitelisted IPs |
| `/admin/whitelist` | POST | Add IP to whitelist |
| `/admin/whitelist/{id}` | DELETE | Remove from whitelist |
| `/admin/tunnels` | GET | List active tunnels |
| `/admin/stats` | GET | Get server statistics |

### Public Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Health check (returns JSON) |
| `/_tunnel` | WebSocket endpoint for tunnel clients |

### Tunnel Auth Endpoints (on subdomains)

| Endpoint | Description |
|----------|-------------|
| `/__auth/login` | Start OIDC login flow |
| `/__auth/callback` | OIDC callback |
| `/__auth/logout` | Clear session |
| `/__auth/health` | Auth system health check |

## License

MIT
