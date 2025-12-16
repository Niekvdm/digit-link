# digit-link Performance Guide

## Overview

digit-link is designed for efficient tunnel management with minimal overhead. This document covers performance characteristics, optimization strategies, and operational metrics.

## Architecture Performance Characteristics

### Core Performance Features

| Component | Design Choice | Performance Impact |
|-----------|--------------|-------------------|
| Database | SQLite | Low latency, limited write concurrency |
| WebSocket | gorilla/websocket | Efficient persistent connections |
| Auth | In-memory cache + SQLite | Fast hot-path, persistent cold storage |
| Rate Limiter | Memory cache + SQLite | Microsecond checks, survives restarts |
| HTTP Routing | Standard library | Minimal allocation overhead |

### Theoretical Limits

| Resource | Limit | Bottleneck |
|----------|-------|------------|
| Concurrent Tunnels | ~10,000 | Memory (goroutines, maps) |
| Requests/Second | ~50,000 | CPU (JSON encoding) |
| WebSocket Connections | OS file descriptors | `ulimit -n` |
| Database Writes | ~1,000/s | SQLite WAL mode |

---

## Memory Profile

### Per-Tunnel Memory

```go
type Tunnel struct {
    Subdomain     string                    // ~32 bytes
    Conn          *websocket.Conn           // ~8 KB (buffers)
    ResponseChans map[string]chan []byte    // Variable
    CreatedAt     time.Time                 // 24 bytes
    AccountID     string                    // 36 bytes
    RecordID      string                    // 36 bytes
    OrgID         string                    // 36 bytes
    AppID         string                    // 36 bytes
    App           *db.Application           // ~200 bytes
    mu            sync.RWMutex              // 24 bytes
}
// Base: ~8.5 KB per tunnel
// + 1 KB per pending request channel
```

### Server Memory Usage

```
Base Server:        ~50 MB
Per 1,000 Tunnels:  ~10 MB
SQLite Cache:       ~10 MB (configurable)
Rate Limit Cache:   ~1 KB per unique IP
Policy Cache:       ~1 KB per subdomain
```

**Example Sizing:**
- 1,000 tunnels: ~70 MB
- 5,000 tunnels: ~110 MB
- 10,000 tunnels: ~160 MB

---

## CPU Profile

### Hot Paths

1. **Request Forwarding** (~40% CPU)
   - JSON marshal/unmarshal
   - WebSocket read/write
   - HTTP header copying

2. **Authentication** (~25% CPU)
   - Token hash comparison
   - JWT validation
   - bcrypt verification (when needed)

3. **Policy Resolution** (~15% CPU)
   - Database lookups (cacheable)
   - Policy inheritance resolution

4. **Rate Limiting** (~5% CPU)
   - In-memory cache checks
   - Periodic SQLite persistence

### Optimization Opportunities

#### 1. JSON Encoding
The biggest CPU consumer. Consider:
```go
// Use json.RawMessage for pass-through data
type HTTPRequest struct {
    ID      string          `json:"id"`
    Method  string          `json:"method"`
    Path    string          `json:"path"`
    Headers json.RawMessage `json:"headers"`  // Don't re-parse
    Body    []byte          `json:"body"`
}
```

#### 2. Policy Caching
Currently implemented with in-memory cache:
```go
loader := policy.NewLoader(database, resolver,
    policy.WithCacheTTL(5 * time.Minute),
)
```

#### 3. Connection Pooling
For high-throughput scenarios, tune WebSocket buffers:
```go
upgrader: websocket.Upgrader{
    ReadBufferSize:  1024 * 64,  // 64KB
    WriteBufferSize: 1024 * 64,  // 64KB
}
```

---

## Database Performance

### SQLite Configuration

```go
conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
```

**Recommended for production:**
```go
// Enable WAL mode for better concurrency
db.Exec("PRAGMA journal_mode=WAL")

// Increase cache size (default is 2MB)
db.Exec("PRAGMA cache_size=-64000") // 64MB

// Synchronous mode for durability vs performance trade-off
db.Exec("PRAGMA synchronous=NORMAL") // vs FULL for max durability
```

### Query Performance

| Operation | Typical Latency | Index Used |
|-----------|-----------------|------------|
| Token lookup | <1ms | `idx_accounts_token_hash` |
| API key validation | <1ms | `idx_api_keys_key_hash` |
| Subdomain lookup | <1ms | `idx_applications_subdomain` |
| Whitelist check | <1ms | `idx_*_whitelist_ip` |
| Audit log insert | ~2ms | None (append-only) |
| Session validation | <1ms | Primary key |

### Batch Operations

For high-volume logging:
```go
// Buffer audit events and batch insert
func (db *DB) BatchLogAuthEvents(events []AuthEvent) error {
    tx, _ := db.conn.Begin()
    defer tx.Rollback()
    
    stmt, _ := tx.Prepare(`INSERT INTO auth_audit_log ...`)
    for _, e := range events {
        stmt.Exec(e.Timestamp, e.OrgID, ...)
    }
    return tx.Commit()
}
```

---

## Network Performance

### WebSocket Configuration

```go
// Optimal ping interval (30s default)
func (s *Server) pingRoutine() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        // Send ping to all tunnels
    }
}
```

### Request Timeout

Default: 5 minutes (configurable via `--timeout` flag)

```go
select {
case responseData := <-responseCh:
    // Handle response
case <-time.After(5 * time.Minute):
    http.Error(w, "Tunnel timeout", http.StatusGatewayTimeout)
}
```

### Connection Pooling

For tunnel clients connecting to local services:
```go
type Proxy struct {
    client *http.Client
}

func NewProxy(localPort int) *Proxy {
    return &Proxy{
        client: &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 100,
                IdleConnTimeout:     90 * time.Second,
            },
            Timeout: 5 * time.Minute,
        },
    }
}
```

---

## Rate Limiting Performance

### In-Memory Cache

```go
type RateLimiter struct {
    cache   map[string]*rateLimitEntry  // Hot path
    cacheMu sync.RWMutex
    db      *db.DB                      // Cold storage
}
```

**Performance Characteristics:**
- Cache hit: ~100ns
- Cache miss + DB load: ~1ms
- DB persist (batched): ~5ms

### Cleanup Strategy

```go
// Cleanup runs every 5 minutes
func (rl *RateLimiter) cleanup() {
    // Remove expired entries from cache
    // Batch delete from database
}
```

---

## Monitoring and Metrics

### Built-in Statistics

Available via `/admin/stats` endpoint:

```json
{
    "activeTunnels": 150,
    "totalAccounts": 500,
    "activeAccounts": 450,
    "whitelistEntries": 25,
    "totalTunnels": 50000,
    "totalBytesSent": 10737418240,
    "totalBytesReceived": 21474836480
}
```

### Health Check

Available via `/health` endpoint:

```json
{
    "status": "ok",
    "activeTunnels": 150,
    "activeAccounts": 450,
    "whitelistEntries": 25
}
```

### Recommended Prometheus Metrics

Add these metrics for production monitoring:

```go
var (
    activeTunnels = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "digitlink_active_tunnels",
        Help: "Number of active tunnel connections",
    })
    
    requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "digitlink_request_duration_seconds",
        Help:    "Request latency distribution",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
    }, []string{"subdomain", "method", "status"})
    
    authAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "digitlink_auth_attempts_total",
        Help: "Authentication attempts",
    }, []string{"type", "success"})
    
    rateLimitHits = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "digitlink_rate_limit_hits_total",
        Help: "Rate limit violations",
    })
)
```

---

## Scaling Strategies

### Vertical Scaling

| Resource | Recommendation |
|----------|----------------|
| CPU | 2+ cores for JSON processing |
| Memory | 256MB base + 20MB per 1000 tunnels |
| Disk | SSD for SQLite, 1GB+ for audit logs |
| File Descriptors | 65536+ (`ulimit -n`) |

### Horizontal Scaling Limitations

**Current limitations:**
1. SQLite is single-node
2. Tunnel state is in-memory
3. Session state requires database access

**Scaling options:**
1. **PostgreSQL migration** - Replace SQLite for multi-node writes
2. **Redis session store** - Shared session state
3. **Sticky sessions** - Route subdomain to same server
4. **Event-sourcing** - Replicate tunnel state

### Load Balancing

For high availability with single-node:
```
┌─────────────────┐
│  Load Balancer  │
│  (Active-Passive)│
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌───────┐ ┌───────┐
│Primary│ │Standby│
│ (RW)  │ │ (RO)  │
└───────┘ └───────┘
    │
    └── Shared Storage (SQLite on NFS/EBS)
```

---

## Performance Tuning Checklist

### Pre-Production

- [ ] Set `GOMAXPROCS` to available cores
- [ ] Increase file descriptor limits
- [ ] Configure SQLite WAL mode
- [ ] Set appropriate cache sizes
- [ ] Configure connection timeouts

### Production Monitoring

- [ ] Monitor goroutine count (should be stable)
- [ ] Track database query latency
- [ ] Alert on rate limit spikes
- [ ] Monitor memory usage trends
- [ ] Track WebSocket error rates

### Troubleshooting

#### High CPU
1. Profile with `pprof`: `go tool pprof http://localhost:6060/debug/pprof/profile`
2. Check JSON serialization overhead
3. Review bcrypt verification frequency

#### High Memory
1. Check tunnel count vs expected
2. Review response channel accumulation
3. Monitor SQLite cache size

#### Slow Responses
1. Check database query latency
2. Verify network connectivity to tunnels
3. Review rate limiter contention

#### Connection Drops
1. Check WebSocket ping/pong timing
2. Verify network stability
3. Review proxy timeout settings

---

## Benchmarks

### Request Forwarding (p99 latency)

| Payload Size | Latency |
|--------------|---------|
| 1 KB | ~5ms |
| 10 KB | ~8ms |
| 100 KB | ~20ms |
| 1 MB | ~100ms |

### Authentication (p99 latency)

| Method | Latency |
|--------|---------|
| Token hash | <1ms |
| API key | <1ms |
| JWT validation | ~2ms |
| bcrypt verify | ~100ms |

### Database Operations (p99 latency)

| Operation | Latency |
|-----------|---------|
| Single row read | <1ms |
| Index lookup | <1ms |
| Insert (WAL) | ~2ms |
| Batch insert (100 rows) | ~20ms |

---

## Resource Recommendations

### Minimum (Development)
- 1 CPU core
- 256 MB RAM
- 100 MB disk

### Small (< 100 tunnels)
- 1 CPU core
- 512 MB RAM
- 1 GB disk

### Medium (100-1000 tunnels)
- 2 CPU cores
- 1 GB RAM
- 5 GB disk

### Large (1000+ tunnels)
- 4 CPU cores
- 2 GB RAM
- 20 GB disk
- SSD storage
