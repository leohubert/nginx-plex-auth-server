# Nginx Plex Auth Server

A lightweight Go-based authentication server for Nginx's `auth_request` module that validates requests against Plex authentication.

## Features

- Validates Plex authentication tokens
- Verifies user access to specific Plex servers
- Works with Nginx `auth_request` directive
- Supports multiple token sources (Authorization header, X-Plex-Token header, cookies)
- Validates both server owners and shared users
- **In-memory cache system** to reduce API calls to Plex (configurable TTL)
- Health check endpoint
- Configurable via environment variables

## Project Structure

```
.
├── cmd/
│   └── server/          # Application entry point
│       └── main.go
├── internal/
│   ├── auth/           # Authentication logic
│   │   ├── handler.go
│   │   └── oauth.go
│   ├── cache/          # Token caching system
│   │   └── token_cache.go
│   ├── config/         # Configuration management
│   │   └── config.go
│   └── middleware/     # HTTP middlewares (future use)
├── pkg/
│   └── plex/          # Plex API client
│       └── client.go
├── go.mod
├── go.sum
└── README.md
```

## Configuration

The server is configured using environment variables:

- `PLEX_TOKEN` (required): Your Plex server owner's authentication token (used to verify server access)
- `PLEX_SERVER_ID` (required): The machine identifier of your Plex server
- `PLEX_CLIENT_ID` (optional): Client identifier for Plex OAuth (defaults to `nginx-plex-auth-server`)
- `PLEX_URL` (optional): Plex API URL (defaults to `https://plex.tv`)
- `SERVER_ADDR` (optional): Server listen address (defaults to `:8080`)
- `CALLBACK_URL` (optional): OAuth callback URL (defaults to `http://localhost:8080/callback`)
- `COOKIE_DOMAIN` (optional): Domain for session cookies (leave empty for current domain)
- `COOKIE_SECURE` (optional): Set to `true` for HTTPS-only cookies (defaults to `false`)
- `CACHE_TTL_SECONDS` (optional): Token cache TTL in seconds (defaults to `300` = 5 minutes)
- `CACHE_MAX_SIZE` (optional): Maximum number of tokens to cache (defaults to `1000`)

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
go build -o bin/auth-server ./cmd/server
```

### Running

```bash
export PLEX_TOKEN="your-plex-owner-token"
export PLEX_SERVER_ID="your-server-machine-id"
export SERVER_ADDR=":8080"
./bin/auth-server
```

Or with Go:

```bash
export PLEX_TOKEN="your-plex-owner-token"
export PLEX_SERVER_ID="your-server-machine-id"
go run ./cmd/server/main.go
```

### Nginx Configuration

Add this to your Nginx configuration:

```nginx
location /protected/ {
    auth_request /auth;
    # Your protected content configuration
}

location = /auth {
    internal;
    proxy_pass http://localhost:8080/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header X-Original-URI $request_uri;
}
```

## Endpoints

### Authentication Endpoints

- `GET /auth` - Nginx auth_request endpoint
  - Returns `200 OK` if user has valid token and access to the server
  - Returns `401 Unauthorized` if token is missing or invalid
  - Returns `403 Forbidden` if user doesn't have access to the specified Plex server
  - Returns `500 Internal Server Error` on API errors

### OAuth Flow Endpoints

- `GET /login` - Initiates Plex OAuth flow, displays login page with PIN
- `GET /callback` - OAuth callback endpoint (called by JavaScript polling)
- `GET /auth-success` - Success page after authentication
- `GET /logout` - Clears session cookie and logs out user
- `GET /status` - Returns JSON with authentication status

### Utility Endpoints

- `GET /health` - Health check endpoint

## Token Caching

To minimize API calls to Plex and improve performance, the server implements an in-memory token cache system:

- **Cache Duration**: Validated tokens are cached for 5 minutes by default (configurable via `CACHE_TTL_SECONDS`)
- **Cache Size**: Up to 1000 tokens can be cached (configurable via `CACHE_MAX_SIZE`)
- **Automatic Cleanup**: Expired entries are automatically removed every minute
- **Cache Invalidation**: Tokens are removed from cache on logout

### Cache Benefits

- **Reduced API Load**: Each cached token eliminates 2-3 API calls to Plex
- **Improved Response Time**: Auth requests are served instantly from cache
- **Better User Experience**: Faster page loads for authenticated users

### Cache Behavior

- First request with a token: Validates with Plex API and caches the result
- Subsequent requests (within TTL): Served from cache
- After TTL expires: Token is revalidated with Plex API and cache is refreshed
- Invalid tokens are also cached to prevent repeated failed API calls

## Authentication & Authorization

The server performs two-step validation:

1. **Token Validation**: Verifies the Plex token is valid
2. **Server Access Check**: Confirms the user has access to the specified Plex server (either as owner or shared user)

The server accepts authentication tokens in the following order of precedence:

1. `Authorization` header (supports `Bearer <token>` format)
2. `X-Plex-Token` header
3. `X-Plex-Token` cookie

### How Server Access Works

- If the authenticating user is the server owner (matches `PLEX_TOKEN`), access is granted
- If the user is not the owner, the server checks if they have shared access to the specified server
- Only users explicitly shared on the Plex server will be granted access

## OAuth Login Flow

This server supports Plex OAuth authentication with automatic session cookie creation and redirect back to the original protected page:

1. User tries to access a protected resource (e.g., `/protected/content`)
2. Nginx `auth_request` sends request to `/auth` endpoint - authentication fails (401)
3. Nginx `error_page 401` redirects to `/login?redirect=https://example.com/protected/content`
4. Server requests a PIN from Plex API
5. User is shown a login page with:
   - A "Login with Plex" button that opens Plex.tv in a popup
   - The PIN code displayed for manual entry if needed
   - Automatic polling to detect when authentication completes
6. User authenticates on Plex.tv in the popup
7. JavaScript polls `/callback` endpoint to check if authentication completed
8. Server verifies user has access to the specified Plex server
9. On success, server creates a session cookie (`X-Plex-Token`) valid for 30 days
10. User is **automatically redirected back to the original protected URL** they were trying to access

### Important Notes:

- The redirect URL is captured from the nginx `error_page` directive
- After successful login, users are sent back to where they originally wanted to go
- If no redirect URL is provided, users are sent to `/` (the welcome page)
- The welcome page shows login status and provides quick access to login/logout

### Nginx Configuration for OAuth

To support user login via browser, add these locations to your Nginx config:

```nginx
# Protected content
location /protected/ {
    auth_request /auth;
    error_page 401 = @error401;
    # Your protected content configuration
}

# Handle unauthorized access - redirect to login with original URL
location @error401 {
    return 302 http://localhost:8080/login?redirect=$scheme://$host$request_uri;
}

# Nginx auth check (internal only)
location = /auth {
    internal;
    proxy_pass http://localhost:8080/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header X-Original-URI $request_uri;
    # Forward cookies to auth server
    proxy_set_header Cookie $http_cookie;
}

# OAuth endpoints (accessible to users)
location ~ ^/(login|callback|auth-success|logout|status)$ {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

### Session Management

- Session cookies are valid for 30 days
- Cookies are HttpOnly for security
- Set `COOKIE_SECURE=true` when using HTTPS
- Set `COOKIE_DOMAIN` to share cookies across subdomains
- Visit `/logout` to clear the session cookie

## Development

### Prerequisites

- Go 1.21 or higher
- A valid Plex account and token

### Running Tests

```bash
go test ./...
```

### Docker Support

#### Building the Docker Image

##### Local Build (Single Architecture)

```bash
docker build -t nginx-plex-auth-server .
```

##### Multi-Platform Build (AMD64 + ARM64)

For pushing images that work on both Intel/AMD and ARM servers:

```bash
# Create a new builder instance (first time only)
docker buildx create --name multiplatform --use

# Build and push for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag leohubert/nginx-plex-auth-server:latest \
  --push \
  .

# Or for GHCR
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ghcr.io/leohubert/nginx-plex-auth-server:latest \
  --push \
  .
```

**Note:** Multi-platform builds with `--push` will automatically push to the registry. Make sure you're logged in first with `docker login`.

#### Running with Docker

```bash
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  -e PLEX_TOKEN="your-plex-owner-token" \
  -e PLEX_SERVER_ID="your-server-machine-id" \
  -e PLEX_CLIENT_ID="your-client-id" \
  -e COOKIE_SECURE="false" \
  nginx-plex-auth-server
```

#### Using Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  plex-auth-server:
    build: .
    container_name: plex-auth-server
    ports:
      - "8080:8080"
    environment:
      - PLEX_TOKEN=your-plex-owner-token
      - PLEX_SERVER_ID=your-server-machine-id
      - PLEX_CLIENT_ID=your-client-id
      - PLEX_URL=https://plex.tv
      - COOKIE_SECURE=false
      - COOKIE_DOMAIN=
      - CACHE_TTL_SECONDS=300
      - CACHE_MAX_SIZE=1000
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

#### Pushing to Docker Registry

##### Docker Hub

```bash
# Login to Docker Hub
docker login

# Tag your image with your Docker Hub username
docker tag nginx-plex-auth-server leohubert/nginx-plex-auth-server:latest
docker tag nginx-plex-auth-server leohubert/nginx-plex-auth-server:v1.0.0

# Push to Docker Hub
docker push leohubert/nginx-plex-auth-server:latest
docker push leohubert/nginx-plex-auth-server:v1.0.0
```

##### GitHub Container Registry (GHCR)

```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u leohubert --password-stdin

# Tag your image for GHCR
docker tag nginx-plex-auth-server ghcr.io/leohubert/nginx-plex-auth-server:latest
docker tag nginx-plex-auth-server ghcr.io/leohubert/nginx-plex-auth-server:v1.0.0

# Push to GHCR
docker push ghcr.io/leohubert/nginx-plex-auth-server:latest
docker push ghcr.io/leohubert/nginx-plex-auth-server:v1.0.0
```

##### Private Registry

```bash
# Login to your private registry
docker login registry.example.com

# Tag your image for private registry
docker tag nginx-plex-auth-server registry.example.com/nginx-plex-auth-server:latest

# Push to private registry
docker push registry.example.com/nginx-plex-auth-server:latest
```

##### Using the Published Image

After pushing, others can pull and run your image:

```bash
# From Docker Hub
docker pull leohubert/nginx-plex-auth-server:latest
docker run -d -p 8080:8080 -e PLEX_TOKEN="token" -e PLEX_SERVER_ID="id" leohubert/nginx-plex-auth-server:latest

# From GHCR
docker pull ghcr.io/leohubert/nginx-plex-auth-server:latest
docker run -d -p 8080:8080 -e PLEX_TOKEN="token" -e PLEX_SERVER_ID="id" ghcr.io/leohubert/nginx-plex-auth-server:latest
```

#### Docker Image Features

- Multi-stage build for minimal image size (46.3MB)
- Non-root user for security
- Built-in health check
- CA certificates included for HTTPS requests to Plex API
- Based on Alpine Linux for small footprint

## License

MIT
