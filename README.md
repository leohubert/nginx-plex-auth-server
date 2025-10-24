# Nginx Plex Auth Server

A lightweight, high-performance Go-based authentication server for Nginx's `auth_request` module that validates requests against Plex authentication. Protect your web services with Plex user authentication and manage access control based on Plex server ownership or sharing.

## Quick Start

```bash
# Clone and setup
git clone https://github.com/leohubert/nginx-plex-auth-server.git
cd nginx-plex-auth-server

# Configure
cp .env.example .env
# Edit .env and set your PLEX_SERVER_ID

# Run
go run . api
```

The server will start on `localhost:8080`. Visit http://localhost:8080 to see the login page.

## Features

- Validates Plex authentication tokens
- Verifies user access to specific Plex servers
- Works with Nginx `auth_request` directive
- Supports multiple token sources (Authorization header, X-Plex-Token header, cookies)
- Validates both server owners and shared users
- **In-memory cache system** to reduce API calls to Plex (configurable TTL)
- Health check endpoint
- Configurable via environment variables
- Structured logging with JSON or console output
- CORS support for cross-origin requests

## Technology Stack

- **Go 1.25**: Modern Go with latest features
- **Templ**: Type-safe HTML templating for Go
- **Gorilla Mux**: Powerful HTTP router and URL matcher
- **Zap**: Blazing fast, structured logging
- **Custom utilities**: Environment variable handling, error utilities, OS signal handling

## Project Structure

```
.
├── main.go              # Application entry point
├── cmd/
│   ├── bootstrap.go    # Application bootstrap and dependency injection
│   └── api.go          # API server command
├── internal/
│   ├── cache/          # Token caching system
│   │   └── token_cache.go
│   ├── plex/           # Plex API client implementation
│   │   ├── client.go
│   │   ├── create_auth_pin.go
│   │   ├── check_auth_pin.go
│   │   ├── check_server_access.go
│   │   └── ...
│   └── server/         # HTTP server and handlers
│       ├── server.go
│       ├── auth.go
│       ├── login.go
│       ├── callback.go
│       ├── logout.go
│       └── views/      # Templ templates
├── pkg/
│   ├── envtb/          # Environment variable utilities
│   ├── logtb/          # Logging utilities
│   ├── errtb/          # Error handling utilities
│   └── ostb/           # OS utilities (signal handling)
├── go.mod
├── go.sum
└── README.md
```

## Configuration

The server is configured using environment variables. All configuration can be set via a `.env` file (see `.env.example`).

### Plex Configuration

- **`PLEX_SERVER_ID`** (required): The machine identifier of your Plex server
- **`PLEX_CLIENT_ID`** (optional): Client identifier for Plex OAuth
  - Default: `nginx-plex-auth-server`
- **`PLEX_URL`** (optional): Plex API URL
  - Default: `https://plex.tv`

### Server Configuration

- **`SERVER_ADDR`** (optional): Server listen address
  - Default: `localhost:8080`
  - Format: `host:port` or `:port`

### Cookie Configuration

- **`COOKIE_DOMAIN`** (optional): Domain for session cookies
  - Default: empty (current domain only)
  - Set to `.example.com` to share cookies across subdomains
- **`COOKIE_SECURE`** (optional): HTTPS-only cookies
  - Default: `false`
  - Set to `true` when using HTTPS

### Cache Configuration

- **`CACHE_TTL`** (optional): Token cache time-to-live
  - Default: `10s`
  - Format: `30s`, `1m`, `5m`, etc.
- **`CACHE_MAX_SIZE`** (optional): Maximum cached tokens
  - Default: `100`

### Logging Configuration

- **`LOG_FORMAT`** (optional): Log output format
  - Default: `json`
  - Options: `json`, `console`
- **`SERVER_ACCESS_LOG`** (optional): Enable HTTP request logging
  - Default: `false`
  - Set to `true` for debugging

### Getting Your Plex Server ID

You can find your Plex server machine identifier by:

1. Opening your Plex Web App
2. Going to Settings > Server > General
3. Looking for the "Machine Identifier" field

Or by querying the Plex API:
```bash
curl -X GET "https://plex.tv/api/v2/resources?includeHttps=1&X-Plex-Token=YOUR_TOKEN"
```

## Usage

### Building

```bash
go build -o bin/auth-server .
```

### Running

```bash
export PLEX_SERVER_ID="your-server-machine-id"
export SERVER_ADDR="localhost:8080"
./bin/auth-server api
```

Or with Go:

```bash
export PLEX_SERVER_ID="your-server-machine-id"
go run . api
```

Or using a `.env` file (recommended):

```bash
cp .env.example .env
# Edit .env with your configuration
go run . api
```

### Nginx Integration

This server is designed to work with Nginx's `auth_request` module. See the complete [Nginx Configuration](#nginx-configuration) section below for a full example including OAuth endpoints.

## API Endpoints

### Authentication Endpoint

**`GET /auth`** - Nginx auth_request validation endpoint (internal use)
- Checks `X-Plex-Token` from header or cookie
- Returns `200 OK` if user has valid token and server access
- Returns `401 Unauthorized` if token is missing or invalid
- Returns `403 Forbidden` if user lacks access to the Plex server
- Uses cache to minimize API calls to Plex

### User-Facing Endpoints

**`GET /`** - Home/Login page
- Shows login interface if not authenticated
- Shows user info and server access status if authenticated
- Supports `?redirect=` query parameter for post-login redirect

**`POST /auth/generate-pin`** - Generate Plex OAuth PIN
- Creates a new Plex PIN for authentication
- Returns JSON: `{"pin_id": 123, "code": "ABCD", "auth_url": "https://..."}`
- Called by JavaScript during login flow

**`GET /callback?pin_id=123`** - Check PIN authentication status
- Polls to check if PIN has been authorized
- Returns `200 OK` with success JSON when authenticated
- Returns `401 Unauthorized` if PIN not yet authorized (keep polling)
- Returns `403 Forbidden` if user lacks server access
- Sets session cookie on successful authentication

**`GET /logout`** - Logout and clear session
- Invalidates cache entry for the user's token
- Clears the `X-Plex-Token` cookie
- Redirects to home page (`/`)

**`GET /health`** - Health check endpoint
- Returns `200 OK` for monitoring/health checks

## Token Caching

To minimize API calls to Plex and improve performance, the server implements an in-memory LRU cache:

### Cache Configuration

- **TTL (Time To Live)**: Configurable via `CACHE_TTL` (default: `10s`)
  - Supports duration format: `30s`, `1m`, `5m`, etc.
- **Max Size**: Configurable via `CACHE_MAX_SIZE` (default: `100` tokens)
  - When full, oldest entry is evicted to make room for new ones
- **Cleanup**: Expired entries are automatically removed every minute

### What Gets Cached

The cache stores validation results for both successful and failed authentications:

- **Valid tokens with access**: Cached with `Valid=true, HasAccess=true`
- **Valid tokens without access**: Cached with `Valid=true, HasAccess=false`
- **Invalid tokens**: Cached with `Valid=false, HasAccess=false`

This prevents repeated API calls for both valid and invalid tokens during the TTL period.

### Cache Lifecycle

1. **First Request**: Token is validated against Plex API, result is cached
2. **Cache Hit** (within TTL): Auth check returns instantly from cache
3. **Cache Miss** (after TTL): Token is revalidated and cache is refreshed
4. **Logout**: Token is immediately invalidated and removed from cache
5. **Eviction**: When cache is full, oldest entry is removed (LRU strategy)

### Benefits

- **Reduced API Load**: Each cache hit eliminates 2 API calls to Plex (user info + server access check)
- **Faster Response Time**: Sub-millisecond auth checks from cache vs 100-500ms API calls
- **Rate Limit Protection**: Helps stay within Plex API rate limits

## Authentication & Authorization

The server performs two-step validation:

1. **Token Validation**: Verifies the Plex token is valid by checking with the Plex API
2. **Server Access Check**: Confirms the user has access to the specified Plex server (either as owner or shared user)

The server accepts authentication tokens from the following sources:

1. `X-Plex-Token` header
2. `X-Plex-Token` cookie (set automatically after successful OAuth login)

### Nginx Configuration

Here's a complete Nginx configuration example:

```nginx
upstream plex_auth_server {
    server localhost:8080;
}

server {
    listen 80;
    server_name example.com;

    # Protected content example
    location /protected/ {
        auth_request /auth;
        error_page 401 = @error401;

        # Your protected content here
        root /var/www/html;
        index index.html;
    }

    # Redirect unauthorized users to login with return URL
    location @error401 {
        return 302 $scheme://$host/?redirect=$scheme://$host$request_uri;
    }
}
```

### Configuration Notes:

- **Cookie Forwarding**: The `proxy_set_header Cookie $http_cookie;` in `/auth` location is **critical** - without it, the auth server cannot read the session cookie
- **Redirect After Login**: The `@error401` handler captures the original URL and passes it via `?redirect=` parameter
- **Internal Location**: The `/auth` endpoint must be marked `internal` to prevent direct external access
- **Upstream**: Use `upstream` block for better connection pooling and load balancing

### Session Management

- Session cookies are valid for 30 days
- Cookies are HttpOnly for security
- Set `COOKIE_SECURE=true` when using HTTPS
- Set `COOKIE_DOMAIN` to share cookies across subdomains
- Visit `/logout` to clear the session cookie

## Development

### Prerequisites

- Go 1.25 or higher
- A valid Plex account
- Access to a Plex server (as owner or shared user)

### Running Tests

```bash
go test ./...
```

### Template Generation

This project uses [Templ](https://templ.guide/) for type-safe HTML templates. If you modify `.templ` files, you need to regenerate the Go code:

```bash
go tool templ generate
```

### Docker Deployment

The Docker image is multi-platform (AMD64/ARM64), uses Alpine Linux for a small footprint (~50MB), runs as non-root user, and includes built-in health checks.

#### Using Docker Compose (Recommended)

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    container_name: plex-auth-server
    ports:
      - "8080:8080"
    environment:
      - PLEX_SERVER_ID=your-server-machine-id
      - PLEX_CLIENT_ID=nginx-plex-auth-server
      - PLEX_URL=https://plex.tv
      - SERVER_ADDR=localhost:8080
      - COOKIE_SECURE=false
      - COOKIE_DOMAIN=
      - CACHE_TTL=10s
      - CACHE_MAX_SIZE=100
      - LOG_FORMAT=json
      - SERVER_ACCESS_LOG=false
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3
```

Run with:

```bash
docker-compose up -d
```

#### Using Docker Run

```bash
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  -e PLEX_SERVER_ID="your-server-machine-id" \
  ghcr.io/leohubert/nginx-plex-auth-server:latest
```

## Troubleshooting

### Authentication Fails (401/403)

1. **Check your Plex Server ID**: Ensure `PLEX_SERVER_ID` matches your actual Plex server's machine identifier
2. **Verify user access**: Make sure the user has access to the specified Plex server (either as owner or shared user)
3. **Check cookies**: Ensure cookies are being forwarded from Nginx to the auth server
4. **Cookie security**: If using HTTPS, set `COOKIE_SECURE=true`

### Debugging Tips

Enable detailed logging:
```bash
export LOG_FORMAT=console
export SERVER_ACCESS_LOG=true
go run . api
```

This will show all HTTP requests and responses in a human-readable format.

## License

MIT
