# Nginx Plex Auth Server

A lightweight authentication server for Nginx that validates requests against Plex authentication. Protect your web services with Plex user authentication.

## Features

- Validates Plex tokens via Nginx's `auth_request` module
- OAuth flow with Plex PIN authentication
- In-memory caching to minimize API calls
- Supports server owners and shared users
- Docker ready with health checks

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

Visit http://localhost:8080 to see the login page.

### Get Your Plex Server ID

1. Open Plex Web App → Settings → Your Server → General
2. Copy the "Machine Identifier" field (you can found it in the URL, e.g. `https://app.plex.tv/desktop/#!/settings/server/<YOUR_SERVER_ID>/settings`)

Or query the API:
```bash
curl "https://plex.tv/api/v2/resources?includeHttps=1&X-Plex-Token=YOUR_TOKEN"
```

## Docker

```bash
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  -e PLEX_SERVER_ID="your-server-machine-id" \
  ghcr.io/leohubert/nginx-plex-auth-server:latest
```

## Basic Nginx Integration

```nginx
upstream plex_auth_server {
    server localhost:8080;
}

server {
    listen 80;
    server_name example.com;

    # Protected content
    location /protected/ {
        auth_request /auth;
        error_page 401 = @error401;

        # Your protected content here
        root /var/www/html;
    }

    # Auth endpoint (internal)
    location /auth {
        internal;
        proxy_pass http://plex_auth_server/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Cookie $http_cookie;  # Required!
    }

    # Redirect to login
    location @error401 {
        return 302 $scheme://$host/?redirect=$scheme://$host$request_uri;
    }

    # Public endpoints
    location ~ ^/(|auth/generate-pin|callback|logout|health)$ {
        proxy_pass http://plex_auth_server;
        proxy_set_header Host $host;
    }
}
```

**Important:** The `Cookie` header in `/auth` location is required for authentication to work.

## Configuration

Required configuration in `.env`:

```bash
PLEX_SERVER_ID=your-server-machine-id
```

Optional configuration:

```bash
SERVER_ADDR=localhost:8080      # Server address
COOKIE_DOMAIN=.example.com      # Share cookies across subdomains
COOKIE_SECURE=true              # HTTPS-only cookies
CACHE_TTL=5m                    # Cache duration
CACHE_MAX_SIZE=500              # Max cached tokens
LOG_FORMAT=json                 # json or console
```

See [Configuration Guide](./docs/configuration.md) for all options.

## How It Works

1. User visits protected page → Nginx sends subrequest to `/auth`
2. Auth server checks token from cookie/header
3. Valid token + server access → Return 200 (access granted)
4. Invalid/missing token → Return 401 (redirect to login)
5. Valid token but no access → Return 403 (access denied)

Results are cached to minimize API calls to Plex.

## Documentation

- **[Configuration Guide](./docs/configuration.md)** - Complete environment variable reference
- **[API Documentation](./docs/api.md)** - All endpoints and authentication flow
- **[Nginx Integration](./docs/nginx.md)** - Comprehensive Nginx setup guide
- **[Docker Deployment](./docs/docker.md)** - Docker Compose and production setups
- **[Architecture](./docs/architecture.md)** - Technical details, caching, and performance

## Development

### Prerequisites

- Go 1.25+
- Plex account with server access

### Building

```bash
# Install dependencies
go mod download

# Build
go build -o bin/auth-server .

# Run
export PLEX_SERVER_ID="your-server-id"
./bin/auth-server api
```

### Template Generation

If you modify `.templ` files:

```bash
go install github.com/a-h/templ/cmd/templ@latest
templ generate
```

### Running Tests

```bash
go test ./...
```

## Troubleshooting

### Authentication fails (401)

- Verify `PLEX_SERVER_ID` is correct
- Check cookies are forwarded: `proxy_set_header Cookie $http_cookie;`
- Enable debug logging: `LOG_FORMAT=console SERVER_ACCESS_LOG=true`

### Users with valid tokens get 403

- User doesn't have access to the Plex server
- Verify user is owner or shared user of the server

### HTTPS issues

- Set `COOKIE_SECURE=true` when using HTTPS
- Ensure Nginx forwards the correct protocol

## Project Structure

```
.
├── main.go              # Entry point
├── cmd/                 # Commands and bootstrap
├── internal/
│   ├── cache/           # Token caching
│   ├── plex/            # Plex API client
│   └── server/          # HTTP handlers and views
└── pkg/                 # Reusable utilities
```

See [Architecture Guide](./docs/architecture.md) for technical details.

## License

MIT
