# Architecture & Technical Details

Deep dive into the technical architecture, technology stack, and implementation details.

## Technology Stack

### Core Technologies

- **Go 1.25** - Modern Go with latest features and performance improvements
- **Templ** - Type-safe HTML templating for Go with compile-time safety
- **Gorilla Mux** - Powerful HTTP router with regex route matching and middleware support
- **Zap** - High-performance, structured logging library

### Custom Utilities

- **envtb** - Environment variable handling with type conversion and defaults
- **logtb** - Logging utilities wrapping Zap for consistent log formatting
- **errtb** - Error handling utilities for consistent error responses
- **ostb** - OS utilities for graceful shutdown and signal handling

## Project Structure

```
.
├── main.go                 # Application entry point with CLI
├── cmd/
│   ├── bootstrap.go       # Dependency injection and app bootstrap
│   └── api.go             # API server command handler
├── internal/
│   ├── cache/             # Token caching system
│   │   └── token_cache.go # LRU cache implementation
│   ├── plex/              # Plex API client
│   │   ├── client.go      # HTTP client with auth
│   │   ├── create_auth_pin.go
│   │   ├── check_auth_pin.go
│   │   ├── check_server_access.go
│   │   ├── check_token.go
│   │   └── types.go       # Plex API response types
│   └── server/            # HTTP server
│       ├── server.go      # Server initialization and routing
│       ├── auth.go        # Auth request handler
│       ├── login.go       # Login page handler
│       ├── callback.go    # OAuth callback handler
│       ├── logout.go      # Logout handler
│       └── views/         # Templ templates
│           ├── login.templ
│           └── ...
├── pkg/                   # Reusable packages
│   ├── envtb/             # Environment utilities
│   ├── logtb/             # Logging utilities
│   ├── errtb/             # Error utilities
│   └── ostb/              # OS utilities
├── go.mod
├── go.sum
└── README.md
```

### Architecture Decisions

#### Why Internal?

The `internal/` directory prevents these packages from being imported by external projects. This is intentional because:
- Code is specific to this application
- No guarantees about API stability
- Allows for breaking changes without versioning concerns

#### Why Separate pkg/?

The `pkg/` directory contains reusable utilities that:
- Could be extracted into separate libraries
- Have no dependencies on `internal/`
- Follow standard Go package conventions

## Caching System

The server implements an in-memory LRU (Least Recently Used) cache to minimize API calls to Plex.

### Cache Architecture

```
┌─────────────────────────────────────────────────┐
│                Request Flow                      │
└─────────────────────────────────────────────────┘

1. Auth Request
   ↓
2. Extract Token (from header/cookie)
   ↓
3. Check Cache
   ├─ Cache Hit → Return cached result (sub-ms)
   └─ Cache Miss ↓
      4. Call Plex API (check token + server access)
         ↓
      5. Store in Cache (with TTL)
         ↓
      6. Return result
```

### Cache Implementation

**File:** `internal/cache/token_cache.go`

**Key Components:**

1. **Cache Entry:**
   ```go
   type CacheEntry struct {
       Valid      bool      // Is token valid?
       HasAccess  bool      // Does user have server access?
       ExpiresAt  time.Time // When does this entry expire?
   }
   ```

2. **Cache Store:**
   ```go
   type TokenCache struct {
       cache     map[string]CacheEntry
       mu        sync.RWMutex  // Concurrent access safety
       ttl       time.Duration
       maxSize   int
       accessLog []string      // LRU tracking
   }
   ```

3. **Operations:**
   - `Get(token)` - Retrieve cached validation result
   - `Set(token, valid, hasAccess)` - Store validation result
   - `Invalidate(token)` - Remove entry (used on logout)
   - `cleanup()` - Remove expired entries (runs every minute)

### Cache Benefits

**Performance:**
- **Without cache:** Every auth request = 2 Plex API calls (~200-500ms)
- **With cache:** Cache hit = sub-millisecond response
- **Improvement:** ~500x faster for cached requests

**API Rate Limiting:**
- Reduces load on Plex API servers
- Prevents rate limiting for high-traffic scenarios
- Example: 100 requests/second → 1 cache miss every 10s (with 10s TTL)

**Cost Reduction:**
- Fewer API calls = lower bandwidth
- Minimal memory footprint (~1KB per cached token)

### Cache Configuration

See [Configuration Guide](./configuration.md) for details on:
- `CACHE_TTL` - How long to cache results
- `CACHE_MAX_SIZE` - Maximum number of tokens to cache

### What Gets Cached

The cache stores validation results for three scenarios:

1. **Valid Token + Access** → `Valid=true, HasAccess=true`
   - User can access protected resources
   - Most common case for authenticated users

2. **Valid Token + No Access** → `Valid=true, HasAccess=false`
   - Token is valid but user doesn't have server access
   - Prevents repeated checks for unauthorized users

3. **Invalid Token** → `Valid=false, HasAccess=false`
   - Token is expired, revoked, or never existed
   - Prevents repeated validation of bad tokens

### Cache Eviction

**LRU Strategy:**
- When cache reaches `CACHE_MAX_SIZE`, oldest entry is removed
- "Oldest" = least recently accessed
- Ensures most active users stay cached

**Expiration:**
- Entries expire after `CACHE_TTL`
- Expired entries removed by background cleanup task (every 1 minute)
- On cache miss, expired entries are also removed immediately

**Manual Invalidation:**
- Logout triggers immediate cache invalidation for that token
- Ensures revoked sessions don't remain cached

## Authentication Flow

### OAuth Flow (Plex PIN)

```
┌─────────┐                ┌──────────────┐                ┌─────────┐
│ Browser │                │ Auth Server  │                │ Plex.tv │
└────┬────┘                └──────┬───────┘                └────┬────┘
     │                             │                             │
     │  1. Click "Login"           │                             │
     ├────────────────────────────>│                             │
     │                             │                             │
     │  2. POST /auth/generate-pin │                             │
     │<────────────────────────────┤                             │
     │  {pin_id, code, auth_url}   │                             │
     │                             │                             │
     │  3. Open auth_url           │                             │
     ├───────────────────────────────────────────────────────────>│
     │                             │                             │
     │  4. User authorizes         │                             │
     │<────────────────────────────────────────────────────────────┤
     │                             │                             │
     │  5. Poll GET /callback?pin_id=...                         │
     ├────────────────────────────>│                             │
     │  401 (not claimed yet)      │                             │
     │<────────────────────────────┤                             │
     │                             │                             │
     │  ... (keep polling)         │                             │
     │                             │                             │
     │  6. Poll again              │                             │
     ├────────────────────────────>│  Check PIN                  │
     │                             ├────────────────────────────>│
     │                             │  {authToken}                │
     │                             │<────────────────────────────┤
     │                             │                             │
     │                             │  Verify server access       │
     │                             ├────────────────────────────>│
     │                             │  {servers: [...]}           │
     │                             │<────────────────────────────┤
     │                             │                             │
     │  200 OK + Set-Cookie        │                             │
     │<────────────────────────────┤                             │
     │                             │                             │
     │  7. Redirect to app         │                             │
     └─────────────────────────────┘                             │
```

### Nginx Auth Request Flow

```
┌─────────┐         ┌───────┐         ┌──────────────┐         ┌─────────┐
│ Browser │         │ Nginx │         │ Auth Server  │         │ Plex.tv │
└────┬────┘         └───┬───┘         └──────┬───────┘         └────┬────┘
     │                  │                     │                      │
     │  GET /protected  │                     │                      │
     ├─────────────────>│                     │                      │
     │                  │                     │                      │
     │                  │  Sub-request: GET /auth                    │
     │                  │  Cookie: X-Plex-Token=abc123               │
     │                  ├────────────────────>│                      │
     │                  │                     │                      │
     │                  │                     │  1. Check Cache      │
     │                  │                     │  ├─ Hit? Return now  │
     │                  │                     │  └─ Miss? ↓          │
     │                  │                     │                      │
     │                  │                     │  2. Validate Token   │
     │                  │                     ├─────────────────────>│
     │                  │                     │  {user info}         │
     │                  │                     │<─────────────────────┤
     │                  │                     │                      │
     │                  │                     │  3. Check Access     │
     │                  │                     ├─────────────────────>│
     │                  │                     │  {servers}           │
     │                  │                     │<─────────────────────┤
     │                  │                     │                      │
     │                  │                     │  4. Cache Result     │
     │                  │                     │                      │
     │                  │  200 OK             │                      │
     │                  │<────────────────────┤                      │
     │                  │                     │                      │
     │  200 OK          │                     │                      │
     │  (protected page)│                     │                      │
     │<─────────────────┤                     │                      │
     │                  │                     │                      │
```

## Error Handling

### Error Response Format

All API endpoints return consistent JSON error responses:

```go
type ErrorResponse struct {
    Error string `json:"error"`
}
```

**Example:**
```json
{
  "error": "Invalid or missing authentication token"
}
```

### HTTP Status Codes

- **200 OK** - Request successful
- **400 Bad Request** - Invalid request (missing parameters, etc.)
- **401 Unauthorized** - Authentication required or token invalid
- **403 Forbidden** - Authenticated but insufficient permissions
- **500 Internal Server Error** - Server-side error (logged)

### Logging

Errors are logged with structured context using Zap:

```go
logger.Error("Failed to validate token",
    zap.String("token", token[:8]+"..."),  // Redacted
    zap.Error(err),
    zap.String("remote_addr", r.RemoteAddr),
)
```

**JSON Format:**
```json
{
  "level": "error",
  "ts": 1699564800.123,
  "caller": "server/auth.go:45",
  "msg": "Failed to validate token",
  "token": "abc12345...",
  "error": "connection timeout",
  "remote_addr": "192.168.1.100"
}
```

## Security Considerations

### Token Security

1. **Cookie Attributes:**
   - `HttpOnly` - Prevents JavaScript access
   - `Secure` - HTTPS only (configurable)
   - `SameSite=Lax` - CSRF protection
   - 30-day expiration

2. **Token Redaction:**
   - Tokens are partially redacted in logs
   - Only first 8 characters shown for debugging

3. **Cache Isolation:**
   - Each token cached separately
   - No shared state between users

### Authentication Verification

**Two-Step Validation:**
1. Token validity (Plex API)
2. Server access check (owner or shared user)

**Why Both?**
- Valid token ≠ server access
- Users can have valid Plex accounts without access to your server
- Prevents unauthorized access from valid Plex users

### Rate Limiting

Currently no built-in rate limiting. Recommended approaches:

1. **Nginx Rate Limiting:**
   ```nginx
   limit_req_zone $binary_remote_addr zone=auth:10m rate=10r/s;

   location /auth {
       limit_req zone=auth burst=20;
       # ...
   }
   ```

2. **Cloudflare/CDN:**
   - DDoS protection
   - Rate limiting
   - WAF rules

## Performance Characteristics

### Benchmarks

**Auth Request (Cache Hit):**
- Latency: < 1ms
- Memory: ~100 bytes
- CPU: Negligible

**Auth Request (Cache Miss):**
- Latency: 100-500ms (Plex API)
- Memory: ~1KB (cache entry)
- CPU: Minimal (HTTP client + JSON parsing)

### Scalability

**Vertical Scaling:**
- CPU: ~0.1% per 100 req/s (mostly idle)
- Memory: ~20MB base + (cache size × 1KB)
- Disk: None (stateless)

**Horizontal Scaling:**
- Stateless design (except cache)
- Can run multiple instances behind load balancer
- Cache is per-instance (not shared)

**Bottleneck:**
- Plex API rate limits
- Mitigated by caching

### Resource Usage

**Typical Production:**
- Memory: 30-50MB
- CPU: 1-5% (idle), 10-20% (under load)
- Network: Minimal (only cache misses)

**Large Deployment (1000 users):**
- Memory: 50-100MB (500-1000 cached tokens)
- CPU: 5-10% (with appropriate cache TTL)
- Cache hit rate: >99% (with 5m TTL)

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/leohubert/nginx-plex-auth-server.git
cd nginx-plex-auth-server

# Install dependencies
go mod download

# Generate templates (if modified .templ files)
go install github.com/a-h/templ/cmd/templ@latest
templ generate

# Build
go build -o bin/auth-server .

# Run
./bin/auth-server api
```

### Running Tests

```bash
go test ./...
```

### Project Dependencies

View dependency graph:
```bash
go mod graph
```

Main dependencies:
- `github.com/gorilla/mux` - HTTP routing
- `github.com/a-h/templ` - Templating
- `go.uber.org/zap` - Logging
- `github.com/joho/godotenv` - .env file loading

## Future Enhancements

Potential improvements:

1. **Distributed Cache:**
   - Redis for shared cache across instances
   - Better horizontal scaling

2. **Metrics/Monitoring:**
   - Prometheus metrics
   - Cache hit/miss rates
   - Request latency

3. **Rate Limiting:**
   - Built-in per-IP rate limiting
   - Token bucket algorithm

4. **Advanced Auth:**
   - Group-based access control
   - Library-level permissions
   - Session management UI
