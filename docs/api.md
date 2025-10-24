# API Documentation

Complete reference for all API endpoints.

## Authentication Endpoint

### `GET /auth`

Nginx auth_request validation endpoint (internal use by Nginx).

**Purpose:** Validates Plex authentication tokens and checks server access.

**Token Sources:**
The endpoint accepts tokens from the following sources (in order of precedence):
1. `X-Plex-Token` header
2. `X-Plex-Token` cookie (set automatically after successful OAuth login)

**Response Codes:**
- `200 OK` - User has valid token and server access
- `401 Unauthorized` - Token is missing or invalid
- `403 Forbidden` - User lacks access to the Plex server

**Cache Behavior:**
Results are cached based on `CACHE_TTL` configuration to minimize API calls to Plex.

**Example Nginx Usage:**
```nginx
location /protected/ {
    auth_request /auth;
    # Your protected content
}

location /auth {
    internal;
    proxy_pass http://localhost:8080/auth;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;
}
```

## User-Facing Endpoints

### `GET /`

Home/Login page.

**Purpose:** Displays login interface or user info based on authentication status.

**Query Parameters:**
- `redirect` (optional) - URL to redirect to after successful login
  - Example: `/?redirect=https://example.com/protected/page`

**Behavior:**
- **Not authenticated:** Shows login interface with Plex OAuth button
- **Authenticated:** Shows user info and server access status

**Response:** HTML page

### `POST /auth/generate-pin`

Generate Plex OAuth PIN for authentication.

**Purpose:** Creates a new Plex PIN that users can authorize via Plex.tv.

**Request:** No body required

**Response:** JSON
```json
{
  "pin_id": 123456,
  "code": "ABCD",
  "auth_url": "https://app.plex.tv/auth#?clientID=...&code=ABCD"
}
```

**Response Fields:**
- `pin_id` - Unique PIN identifier (used for polling)
- `code` - 4-character PIN code shown to user
- `auth_url` - URL to open for Plex authorization

**Flow:**
1. Frontend calls this endpoint
2. User is redirected to `auth_url`
3. User authorizes on Plex.tv
4. Frontend polls `/callback` with `pin_id`

**Example JavaScript:**
```javascript
const response = await fetch('/auth/generate-pin', { method: 'POST' });
const { pin_id, auth_url } = await response.json();
window.open(auth_url, '_blank');
// Start polling /callback?pin_id=${pin_id}
```

### `GET /callback`

Check PIN authentication status (polling endpoint).

**Purpose:** Checks if a PIN has been authorized by the user.

**Query Parameters:**
- `pin_id` (required) - The PIN ID from `/auth/generate-pin`

**Response Codes:**
- `200 OK` - PIN authorized, user authenticated
  - Sets `X-Plex-Token` cookie
  - Returns JSON: `{"success": true, "redirect": "/protected/page"}`
- `401 Unauthorized` - PIN not yet authorized (keep polling)
  - Returns JSON: `{"error": "PIN not claimed yet"}`
- `403 Forbidden` - PIN authorized but user lacks server access
  - Returns JSON: `{"error": "User does not have access to the Plex server"}`

**Cookie Set on Success:**
- Name: `X-Plex-Token`
- Value: User's Plex authentication token
- Max-Age: 30 days
- HttpOnly: true
- Secure: Based on `COOKIE_SECURE` config
- Domain: Based on `COOKIE_DOMAIN` config

**Polling Pattern:**
```javascript
async function pollCallback(pinId, maxAttempts = 60) {
  for (let i = 0; i < maxAttempts; i++) {
    const response = await fetch(`/callback?pin_id=${pinId}`);

    if (response.ok) {
      const data = await response.json();
      // Authenticated! Redirect if needed
      window.location.href = data.redirect || '/';
      return;
    }

    if (response.status === 403) {
      // Access denied
      alert('You do not have access to this Plex server');
      return;
    }

    // 401 - Keep polling
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
}
```

### `GET /logout`

Logout and clear session.

**Purpose:** Invalidates the user's session and clears authentication.

**Actions:**
1. Invalidates cache entry for the user's token
2. Clears the `X-Plex-Token` cookie
3. Redirects to home page (`/`)

**Response:** HTTP 302 redirect to `/`

**Cookie Clearing:**
Sets `X-Plex-Token` cookie with:
- Value: empty
- Max-Age: -1 (expires immediately)

### `GET /health`

Health check endpoint.

**Purpose:** Simple health check for monitoring and load balancers.

**Response:** `200 OK` with body `OK`

**Example Usage:**
```bash
curl http://localhost:8080/health
# Response: OK
```

## Authentication Flow

### Complete OAuth Flow

1. **User visits protected page** → Nginx returns 401 → Redirect to `/?redirect=...`
2. **User clicks "Login with Plex"** → JavaScript calls `POST /auth/generate-pin`
3. **Server generates PIN** → Returns `pin_id`, `code`, `auth_url`
4. **JavaScript opens `auth_url`** → User authorizes on Plex.tv
5. **JavaScript polls `GET /callback?pin_id=...`** → Server checks if PIN claimed
6. **User authorizes PIN** → Next poll returns 200 with cookie set
7. **JavaScript redirects to `redirect` URL** → User can now access protected content

### Token Validation Flow

1. **Nginx receives request** → Sends subrequest to `/auth`
2. **Auth server checks cache** → If cached and valid, return immediately
3. **If not cached** → Validate with Plex API:
   - Check token validity (user info endpoint)
   - Check server access (resources endpoint)
4. **Cache result** → Store validation result for `CACHE_TTL`
5. **Return response** → 200 (access), 401 (invalid token), or 403 (no access)

## Error Responses

All endpoints return JSON error responses with appropriate HTTP status codes.

**Example Error Response:**
```json
{
  "error": "Description of what went wrong"
}
```

**Common Errors:**
- `400 Bad Request` - Missing required parameters
- `401 Unauthorized` - Authentication required or invalid token
- `403 Forbidden` - Valid authentication but insufficient permissions
- `500 Internal Server Error` - Server-side error (check logs)
