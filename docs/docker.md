# Docker Deployment Guide

Complete guide for deploying the Plex Auth Server using Docker.

## Overview

The official Docker image is:
- **Multi-platform:** AMD64 and ARM64 support
- **Lightweight:** Based on Alpine Linux (~50MB)
- **Secure:** Runs as non-root user
- **Health-checked:** Built-in health check endpoint

**Image:** `ghcr.io/leohubert/nginx-plex-auth-server:latest`

## Quick Start

### Using Docker Compose (Recommended)

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
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3
```

Run:

```bash
docker-compose up -d
```

### Using Docker Run

```bash
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  -e PLEX_SERVER_ID="your-server-machine-id" \
  ghcr.io/leohubert/nginx-plex-auth-server:latest
```

## Configuration

### Environment Variables

All configuration is done via environment variables. See the [Configuration Guide](./configuration.md) for details.

**Minimal Configuration:**
```yaml
environment:
  - PLEX_SERVER_ID=your-server-machine-id
```

**Production Configuration:**
```yaml
environment:
  # Required
  - PLEX_SERVER_ID=your-server-machine-id

  # Server
  - SERVER_ADDR=:8080

  # Cookies (for HTTPS and subdomains)
  - COOKIE_DOMAIN=.example.com
  - COOKIE_SECURE=true

  # Performance
  - CACHE_TTL=5m
  - CACHE_MAX_SIZE=500

  # Logging
  - LOG_FORMAT=json
  - SERVER_ACCESS_LOG=false
```

### Using Environment File

Create a `.env` file:

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

**Docker Compose:**
```yaml
services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    env_file: .env
    ports:
      - "8080:8080"
```

**Docker Run:**
```bash
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  --env-file .env \
  ghcr.io/leohubert/nginx-plex-auth-server:latest
```

## Health Checks

The image includes built-in health check support.

### Docker Compose Health Check

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s      # Check every 30 seconds
  timeout: 3s        # Timeout after 3 seconds
  start_period: 5s   # Grace period on startup
  retries: 3         # Retry 3 times before marking unhealthy
```

### Check Health Status

```bash
# View health status
docker inspect --format='{{.State.Health.Status}}' plex-auth-server

# View health logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' plex-auth-server
```

## Networking

### Bridge Network (Default)

```yaml
services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    ports:
      - "8080:8080"  # Expose to host
```

### Custom Network

```yaml
networks:
  web:
    external: true

services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    networks:
      - web
    expose:
      - "8080"  # Internal only, no host port mapping
```

### With Nginx

```yaml
version: '3.8'

networks:
  web:

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/ssl:ro
    networks:
      - web
    depends_on:
      - plex-auth-server

  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    environment:
      - PLEX_SERVER_ID=your-server-machine-id
      - COOKIE_DOMAIN=.example.com
      - COOKIE_SECURE=true
    networks:
      - web
    expose:
      - "8080"
```

In `nginx.conf`:
```nginx
upstream plex_auth_server {
    server plex-auth-server:8080;  # Use container name
}
```

## Logging

### View Logs

```bash
# Follow logs
docker-compose logs -f plex-auth-server

# Last 100 lines
docker-compose logs --tail=100 plex-auth-server

# With docker run
docker logs -f plex-auth-server
```

### JSON Logs (Production)

```yaml
environment:
  - LOG_FORMAT=json
  - SERVER_ACCESS_LOG=false
```

Example output:
```json
{"level":"info","ts":1699564800.123,"caller":"server/server.go:45","msg":"Server starting","addr":":8080"}
```

### Console Logs (Development)

```yaml
environment:
  - LOG_FORMAT=console
  - SERVER_ACCESS_LOG=true
```

Example output:
```
2024-11-09T10:00:00.123Z  INFO  server/server.go:45  Server starting  {"addr": ":8080"}
```

## Resource Limits

### Set Memory and CPU Limits

```yaml
services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    deploy:
      resources:
        limits:
          cpus: '0.5'      # Max 50% of one CPU core
          memory: 128M     # Max 128MB RAM
        reservations:
          cpus: '0.25'     # Reserve 25% of one CPU core
          memory: 64M      # Reserve 64MB RAM
```

**Typical Resource Usage:**
- **Memory:** 20-50MB (depending on cache size)
- **CPU:** Minimal, mostly idle except during auth requests

## Volumes

The application is stateless and doesn't require persistent volumes. All state is stored in memory (cache) or managed by Plex.

## Updates

### Update to Latest Version

**Docker Compose:**
```bash
docker-compose pull
docker-compose up -d
```

**Docker Run:**
```bash
docker pull ghcr.io/leohubert/nginx-plex-auth-server:latest
docker stop plex-auth-server
docker rm plex-auth-server
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  -e PLEX_SERVER_ID="your-server-machine-id" \
  ghcr.io/leohubert/nginx-plex-auth-server:latest
```

### Auto-Updates with Watchtower

```yaml
version: '3.8'

services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    # ... your config

  watchtower:
    image: containrrr/watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --interval 3600  # Check every hour
```

## Production Example

Complete production-ready `docker-compose.yml`:

```yaml
version: '3.8'

networks:
  web:
    driver: bridge

services:
  plex-auth-server:
    image: ghcr.io/leohubert/nginx-plex-auth-server:latest
    container_name: plex-auth-server
    restart: unless-stopped

    # Environment
    environment:
      - PLEX_SERVER_ID=${PLEX_SERVER_ID}
      - PLEX_CLIENT_ID=nginx-plex-auth-server
      - SERVER_ADDR=:8080
      - COOKIE_DOMAIN=${COOKIE_DOMAIN}
      - COOKIE_SECURE=true
      - CACHE_TTL=5m
      - CACHE_MAX_SIZE=500
      - LOG_FORMAT=json
      - SERVER_ACCESS_LOG=false

    # Networking
    networks:
      - web
    expose:
      - "8080"

    # Health check
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3

    # Resource limits
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          cpus: '0.25'
          memory: 64M

    # Security
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp

  nginx:
    image: nginx:alpine
    container_name: nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/ssl:ro
    networks:
      - web
    depends_on:
      plex-auth-server:
        condition: service_healthy
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs plex-auth-server

# Common issues:
# 1. Missing PLEX_SERVER_ID
# 2. Port 8080 already in use
# 3. Invalid environment variable format
```

### Health Check Failing

```bash
# Test health endpoint manually
docker exec plex-auth-server wget -q -O- http://localhost:8080/health

# Check if server is listening
docker exec plex-auth-server netstat -ln | grep 8080
```

### Can't Connect from Nginx

```bash
# Verify they're on the same network
docker network inspect web

# Test connection from nginx container
docker exec nginx wget -q -O- http://plex-auth-server:8080/health
```

### Permission Issues

The container runs as non-root user `appuser` (UID 1000). If you encounter permission issues with volumes:

```yaml
services:
  plex-auth-server:
    user: "1000:1000"  # Explicitly set user
```

## Building Custom Image

If you want to build from source:

```bash
# Clone repository
git clone https://github.com/leohubert/nginx-plex-auth-server.git
cd nginx-plex-auth-server

# Build image
docker build -t plex-auth-server:custom .

# Run your custom image
docker run -d \
  --name plex-auth-server \
  -p 8080:8080 \
  -e PLEX_SERVER_ID="your-server-machine-id" \
  plex-auth-server:custom
```
