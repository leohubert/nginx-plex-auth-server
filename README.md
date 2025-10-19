# Nginx Plex Auth Server

A lightweight Go-based authentication server for Nginx's `auth_request` module that validates requests against Plex authentication.

## Features

- Validates Plex authentication tokens
- Verifies user access to specific Plex servers
- Works with Nginx `auth_request` directive
- Supports multiple token sources (Authorization header, X-Plex-Token header, cookies)
- Validates both server owners and shared users
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
│   │   └── handler.go
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

This server supports Plex OAuth authentication with automatic session cookie creation:

1. User visits `/login` to start authentication
2. Server requests a PIN from Plex API
3. User is shown a login page with:
   - A "Login with Plex" button that opens Plex.tv in a new tab
   - The PIN code displayed for manual entry if needed
   - Automatic polling to detect when authentication completes
4. User authenticates on Plex.tv
5. JavaScript polls `/callback` endpoint to check if authentication completed
6. Server verifies user has access to the specified Plex server
7. On success, server creates a session cookie (`X-Plex-Token`) valid for 30 days
8. User is redirected to success page

### Nginx Configuration for OAuth

To support user login via browser, add these locations to your Nginx config:

```nginx
# Protected content
location /protected/ {
    auth_request /auth;
    error_page 401 = @error401;
    # Your protected content configuration
}

# Handle unauthorized access - redirect to login
location @error401 {
    return 302 http://localhost:8080/login;
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

```bash
docker build -t nginx-plex-auth-server .
```

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
docker tag nginx-plex-auth-server your-username/nginx-plex-auth-server:latest
docker tag nginx-plex-auth-server your-username/nginx-plex-auth-server:v1.0.0

# Push to Docker Hub
docker push your-username/nginx-plex-auth-server:latest
docker push your-username/nginx-plex-auth-server:v1.0.0
```

##### GitHub Container Registry (GHCR)

```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u your-github-username --password-stdin

# Tag your image for GHCR
docker tag nginx-plex-auth-server ghcr.io/your-github-username/nginx-plex-auth-server:latest
docker tag nginx-plex-auth-server ghcr.io/your-github-username/nginx-plex-auth-server:v1.0.0

# Push to GHCR
docker push ghcr.io/your-github-username/nginx-plex-auth-server:latest
docker push ghcr.io/your-github-username/nginx-plex-auth-server:v1.0.0
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
docker pull your-username/nginx-plex-auth-server:latest
docker run -d -p 8080:8080 -e PLEX_TOKEN="token" -e PLEX_SERVER_ID="id" your-username/nginx-plex-auth-server:latest

# From GHCR
docker pull ghcr.io/your-github-username/nginx-plex-auth-server:latest
docker run -d -p 8080:8080 -e PLEX_TOKEN="token" -e PLEX_SERVER_ID="id" ghcr.io/your-github-username/nginx-plex-auth-server:latest
```

#### Docker Image Features

- Multi-stage build for minimal image size (46.3MB)
- Non-root user for security
- Built-in health check
- CA certificates included for HTTPS requests to Plex API
- Based on Alpine Linux for small footprint

## License

MIT
