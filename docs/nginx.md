# Nginx Integration Guide

Complete guide for integrating the Plex Auth Server with Nginx using the `auth_request` module.

## Overview

This server is designed to work with Nginx's `auth_request` directive, which allows you to protect any location block by validating requests against an external authentication service.

## Basic Configuration

### Minimal Example

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
        index index.html;
    }

    # Auth endpoint (internal only)
    location /auth {
        internal;
        proxy_pass http://plex_auth_server/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Cookie $http_cookie;
    }

    # Redirect to login page
    location @error401 {
        return 302 $scheme://$host/?redirect=$scheme://$host$request_uri;
    }

    # Proxy login/callback/logout endpoints
    location ~ ^/(|auth/generate-pin|callback|logout|health)$ {
        proxy_pass http://plex_auth_server;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Configuration Breakdown

### 1. Upstream Block

```nginx
upstream plex_auth_server {
    server localhost:8080;
    keepalive 32;  # Optional: connection pooling
}
```

**Benefits:**
- Better connection pooling
- Easy to add multiple backend servers
- Centralized backend configuration

### 2. Auth Request Location

```nginx
location /auth {
    internal;
    proxy_pass http://plex_auth_server/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;
}
```

**Important Details:**

- **`internal;`** - Prevents direct external access to the auth endpoint
- **`proxy_pass_request_body off;`** - Don't send request body to auth server
- **`proxy_set_header Content-Length "";`** - Clear content-length header
- **`proxy_set_header Cookie $http_cookie;`** - **CRITICAL:** Forward cookies to auth server

**Common Mistake:** Forgetting the Cookie header means the auth server cannot read the session cookie, causing authentication to fail.

### 3. Protected Locations

```nginx
location /protected/ {
    auth_request /auth;
    error_page 401 = @error401;

    # Your protected content
    proxy_pass http://backend_service;
}
```

**How it works:**
1. Request comes in for `/protected/page`
2. Nginx makes subrequest to `/auth`
3. If auth returns 200, request proceeds
4. If auth returns 401, jump to `@error401`
5. If auth returns 403, return 403 to client

### 4. Error Handler

```nginx
location @error401 {
    return 302 $scheme://$host/?redirect=$scheme://$host$request_uri;
}
```

**Explanation:**
- Redirects to login page with original URL as `redirect` parameter
- After login, user is redirected back to original page

**Example:**
- User visits: `https://example.com/protected/page`
- Redirects to: `https://example.com/?redirect=https://example.com/protected/page`
- After login: Redirects back to `https://example.com/protected/page`

### 5. Public Endpoints

```nginx
location ~ ^/(|auth/generate-pin|callback|logout|health)$ {
    proxy_pass http://plex_auth_server;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**Endpoints:**
- `/` - Login page
- `/auth/generate-pin` - OAuth PIN generation
- `/callback` - OAuth callback
- `/logout` - Logout
- `/health` - Health check

## Advanced Configurations

### HTTPS Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Protected content
    location /protected/ {
        auth_request /auth;
        error_page 401 = @error401;

        proxy_pass http://backend;
    }

    # Auth endpoint
    location /auth {
        internal;
        proxy_pass http://plex_auth_server/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Cookie $http_cookie;
    }

    # Error handler
    location @error401 {
        return 302 https://$host/?redirect=https://$host$request_uri;
    }

    # Public endpoints
    location ~ ^/(|auth/generate-pin|callback|logout|health)$ {
        proxy_pass http://plex_auth_server;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

**Important:** Set `COOKIE_SECURE=true` in your auth server config when using HTTPS.

### Multiple Protected Services

```nginx
upstream plex_auth_server {
    server localhost:8080;
}

upstream app1 {
    server localhost:3000;
}

upstream app2 {
    server localhost:4000;
}

server {
    listen 80;
    server_name example.com;

    # Protect app1
    location /app1/ {
        auth_request /auth;
        error_page 401 = @error401;
        proxy_pass http://app1/;
    }

    # Protect app2
    location /app2/ {
        auth_request /auth;
        error_page 401 = @error401;
        proxy_pass http://app2/;
    }

    # Single auth endpoint for all protected locations
    location /auth {
        internal;
        proxy_pass http://plex_auth_server/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Cookie $http_cookie;
    }

    location @error401 {
        return 302 $scheme://$host/?redirect=$scheme://$host$request_uri;
    }

    location ~ ^/(|auth/generate-pin|callback|logout|health)$ {
        proxy_pass http://plex_auth_server;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Subdomain Configuration

```nginx
# Auth server domain
server {
    listen 80;
    server_name auth.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# Protected app domain
server {
    listen 80;
    server_name app.example.com;

    location / {
        auth_request /auth;
        error_page 401 = @error401;
        proxy_pass http://localhost:3000;
    }

    location /auth {
        internal;
        proxy_pass http://localhost:8080/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Cookie $http_cookie;
    }

    location @error401 {
        return 302 https://auth.example.com/?redirect=$scheme://$host$request_uri;
    }
}
```

**Important:** Set `COOKIE_DOMAIN=.example.com` to share cookies across subdomains.

## Testing

### Test Auth Endpoint

```bash
# Without token (should return 401)
curl -I http://localhost:8080/auth

# With valid token (should return 200)
curl -I -H "X-Plex-Token: YOUR_TOKEN" http://localhost:8080/auth
```

### Test via Nginx

```bash
# Should redirect to login
curl -L http://example.com/protected/

# Should show login page
curl http://example.com/
```

## Troubleshooting

### Authentication Always Fails (401)

**Check:**
1. Verify cookies are being forwarded: `proxy_set_header Cookie $http_cookie;`
2. Check auth server logs: `SERVER_ACCESS_LOG=true`
3. Test auth endpoint directly with token header

**Debug:**
```nginx
location /auth {
    internal;
    proxy_pass http://plex_auth_server/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;

    # Add debug headers
    add_header X-Debug-Cookie $http_cookie always;
}
```

### Redirect Loop

**Cause:** Login page itself is protected

**Solution:** Ensure `/` is publicly accessible:
```nginx
location = / {
    proxy_pass http://plex_auth_server;
}

location /protected/ {
    auth_request /auth;
    # ...
}
```

### HTTPS Mixed Content

**Cause:** Not forwarding HTTPS protocol to auth server

**Solution:**
```nginx
location ~ ^/(|auth/generate-pin|callback|logout)$ {
    proxy_pass http://plex_auth_server;
    proxy_set_header X-Forwarded-Proto $scheme;  # Important!
}
```

### Cookies Not Persisting

**Check:**
1. `COOKIE_DOMAIN` matches your domain structure
2. `COOKIE_SECURE=true` when using HTTPS
3. Browser isn't blocking cookies

## Performance Optimization

### Enable Caching

Auth responses can be cached by Nginx for extremely high-traffic scenarios:

```nginx
proxy_cache_path /var/cache/nginx/auth levels=1:2 keys_zone=auth_cache:10m max_size=100m inactive=60m;

location /auth {
    internal;
    proxy_pass http://plex_auth_server/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;

    # Cache successful auth for 10s
    proxy_cache auth_cache;
    proxy_cache_key "$http_cookie";
    proxy_cache_valid 200 10s;
    proxy_cache_valid 401 403 1s;
}
```

**Note:** The auth server already has built-in caching. Nginx caching is only beneficial for extremely high request rates.

### Connection Pooling

```nginx
upstream plex_auth_server {
    server localhost:8080;
    keepalive 32;
    keepalive_requests 100;
    keepalive_timeout 60s;
}
```

## Complete Production Example

```nginx
upstream plex_auth_server {
    server localhost:8080;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;

    # Protected content
    location /protected/ {
        auth_request /auth;
        error_page 401 = @error401;

        proxy_pass http://backend_service;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }

    # Auth endpoint
    location /auth {
        internal;
        proxy_pass http://plex_auth_server/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Cookie $http_cookie;
    }

    # Error handler
    location @error401 {
        return 302 https://$host/?redirect=https://$host$request_uri;
    }

    # Public auth endpoints
    location ~ ^/(|auth/generate-pin|callback|logout|health)$ {
        proxy_pass http://plex_auth_server;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}

# HTTP to HTTPS redirect
server {
    listen 80;
    server_name example.com;
    return 301 https://$host$request_uri;
}
```
