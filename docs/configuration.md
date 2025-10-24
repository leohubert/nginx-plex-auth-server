# Configuration Guide

The server is configured using environment variables. All configuration can be set via a `.env` file (see `.env.example`).

## Plex Configuration

### `PLEX_SERVER_ID` (required)

The machine identifier of your Plex server.

**Getting Your Plex Server ID:**

You can find your Plex server machine identifier by:

1. Opening your Plex Web App
2. Going to Settings > Server > General
3. Looking for the "Machine Identifier" field

Or by querying the Plex API:
```bash
curl -X GET "https://plex.tv/api/v2/resources?includeHttps=1&X-Plex-Token=YOUR_TOKEN"
```

### `PLEX_CLIENT_ID` (optional)

Client identifier for Plex OAuth.

- **Default:** `nginx-plex-auth-server`
- **Format:** Any string identifying your application

### `PLEX_URL` (optional)

Plex API URL.

- **Default:** `https://plex.tv`
- **Format:** Full URL with protocol

## Server Configuration

### `SERVER_ADDR` (optional)

Server listen address.

- **Default:** `localhost:8080`
- **Format:** `host:port` or `:port`
- **Examples:**
  - `localhost:8080` - Listen only on localhost
  - `:8080` - Listen on all interfaces
  - `0.0.0.0:8080` - Explicitly listen on all interfaces

## Cookie Configuration

### `COOKIE_DOMAIN` (optional)

Domain for session cookies.

- **Default:** empty (current domain only)
- **Format:** Domain name, optionally starting with `.`
- **Examples:**
  - `` (empty) - Cookie only valid for current domain
  - `.example.com` - Share cookies across all subdomains (auth.example.com, app.example.com, etc.)
  - `example.com` - Cookie valid only for exact domain

### `COOKIE_SECURE` (optional)

Enable HTTPS-only cookies.

- **Default:** `false`
- **Options:** `true`, `false`
- **Important:** Set to `true` when using HTTPS in production for security

## Cache Configuration

The server uses an in-memory LRU cache to reduce API calls to Plex and improve performance.

### `CACHE_TTL` (optional)

Token cache time-to-live - how long validation results are cached.

- **Default:** `10s`
- **Format:** Duration string (`30s`, `1m`, `5m`, `1h`, etc.)
- **Examples:**
  - `10s` - Cache for 10 seconds
  - `1m` - Cache for 1 minute
  - `5m` - Cache for 5 minutes
- **Recommendation:**
  - Development: `10s` to `30s` for quick testing
  - Production: `1m` to `5m` for balance between performance and freshness

### `CACHE_MAX_SIZE` (optional)

Maximum number of tokens to cache.

- **Default:** `100`
- **Format:** Integer
- **Recommendation:**
  - Small deployments (< 10 users): `50`-`100`
  - Medium deployments (10-50 users): `100`-`500`
  - Large deployments (> 50 users): `500`-`1000`

When the cache is full, the oldest entry is evicted to make room for new ones (LRU strategy).

## Logging Configuration

### `LOG_FORMAT` (optional)

Log output format.

- **Default:** `json`
- **Options:**
  - `json` - Structured JSON logs (recommended for production)
  - `console` - Human-readable colored logs (recommended for development)

### `SERVER_ACCESS_LOG` (optional)

Enable HTTP request logging.

- **Default:** `false`
- **Options:** `true`, `false`
- **Note:** Enable for debugging, but disable in production to reduce log volume

## Example Configuration

### Development `.env`

```bash
PLEX_SERVER_ID=your-server-machine-id
SERVER_ADDR=localhost:8080
COOKIE_SECURE=false
CACHE_TTL=10s
CACHE_MAX_SIZE=50
LOG_FORMAT=console
SERVER_ACCESS_LOG=true
```

### Production `.env`

```bash
PLEX_SERVER_ID=your-server-machine-id
PLEX_CLIENT_ID=nginx-plex-auth-server
SERVER_ADDR=:8080
COOKIE_DOMAIN=.example.com
COOKIE_SECURE=true
CACHE_TTL=5m
CACHE_MAX_SIZE=500
LOG_FORMAT=json
SERVER_ACCESS_LOG=false
```
