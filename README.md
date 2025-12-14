# digit-link

A lightweight ngrok-like tunnel system for exposing local services to the internet.

## Architecture

- **Server**: Runs on k3s, handles WebSocket connections from clients and routes HTTP traffic
- **Client**: CLI tool that connects to the server and forwards requests to local ports

## Quick Start

### Client Usage

```bash
# Build the client
make build-client

# Connect to tunnel server
./build/bin/digit-link --server tunnel.digit.zone --subdomain myapp --port 3000

# Your local service is now available at:
# https://myapp.tunnel.digit.zone
```

### Options

```
--server     Tunnel server address (default: tunnel.digit.zone)
--subdomain  Subdomain to register (required)
--port       Local port to forward to (required)
--secret     Server secret for authentication (optional)
```

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

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DOMAIN` | Base domain for tunnels | `tunnel.digit.zone` |
| `SECRET` | Shared secret for authentication | (none) |

### DNS Setup

Point your wildcard DNS to the k3s ingress:

```
*.tunnel    A    <K3S_INGRESS_IP>
tunnel      A    <K3S_INGRESS_IP>
```

## How It Works

1. Client connects to server via WebSocket at `wss://tunnel.digit.zone/_tunnel`
2. Client registers a subdomain (e.g., `myapp`)
3. Server confirms and provides public URL (`https://myapp.tunnel.digit.zone`)
4. When requests arrive at the public URL, server forwards them to the client over WebSocket
5. Client forwards requests to the local service and returns responses

## License

MIT
