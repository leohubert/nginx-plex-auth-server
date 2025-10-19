package auth

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/hubert_i/nginx_plex_auth_server/internal/cache"
	"github.com/hubert_i/nginx_plex_auth_server/internal/config"
	"github.com/hubert_i/nginx_plex_auth_server/pkg/plex"
)

// OAuthHandler manages OAuth authentication flow
type OAuthHandler struct {
	config     *config.Config
	plexClient *plex.Client
	tokenCache *cache.TokenCache
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(cfg *config.Config, client *plex.Client) *OAuthHandler {
	return &OAuthHandler{
		config:     cfg,
		plexClient: client,
		tokenCache: cache.NewTokenCache(cfg.CacheTTL, cfg.CacheMaxSize),
	}
}

// HandleLogin initiates the Plex OAuth flow
func (h *OAuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Get the redirect URL from query parameter (where user was trying to go)
	// This can come from:
	// 1. 'redirect' query parameter (passed by nginx error page)
	// 2. 'rd' query parameter (short form, compatible with other auth systems)
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = r.URL.Query().Get("rd")
	}
	if redirectURL == "" {
		// Try to get from Referer header as fallback
		referer := r.Header.Get("Referer")
		if referer != "" {
			redirectURL = referer
		}
	}
	if redirectURL == "" {
		redirectURL = "/" // Default to home if no redirect specified
	}

	log.Printf("Login initiated with redirect URL: %s", redirectURL)

	// Request a PIN from Plex
	pinResp, err := h.plexClient.RequestAuthPin()
	if err != nil {
		log.Printf("Error requesting auth PIN: %v", err)
		http.Error(w, "Failed to initiate authentication", http.StatusInternalServerError)
		return
	}

	log.Printf("Generated auth PIN: %s (ID: %d)", pinResp.Code, pinResp.ID)

	// Build the Plex.tv authentication URL matching Overseerr's format
	// This must match exactly what Plex expects for OAuth flow
	authURL := fmt.Sprintf("%s/auth/#!?clientID=%s&context[device][product]=%s&context[device][version]=%s&context[device][platform]=%s&context[device][platformVersion]=%s&context[device][device]=%s&context[device][deviceName]=%s&context[device][model]=%s&context[device][layout]=%s&code=%s",
		"https://app.plex.tv",
		h.config.PlexClientID,
		"Nginx+Auth+Server",
		"1.0",
		"Web",
		"1.0",
		"Linux",
		"Nginx+Auth+Server",
		"Plex+OAuth",
		"desktop",
		pinResp.Code,
	)

	// Render the login page with the auth URL, PIN ID, and redirect URL
	h.renderLoginPage(w, authURL, pinResp.ID, pinResp.Code, redirectURL)
}

// HandleCallback handles the OAuth callback and creates a session cookie
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Get the PIN ID from query params
	pinIDStr := r.URL.Query().Get("pin_id")
	if pinIDStr == "" {
		http.Error(w, "Missing pin_id parameter", http.StatusBadRequest)
		return
	}

	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(w, "Invalid pin_id parameter", http.StatusBadRequest)
		return
	}

	// Check the PIN status
	log.Printf("Checking PIN %d status...", pinID)
	checkResp, err := h.plexClient.CheckAuthPin(pinID)
	if err != nil {
		log.Printf("Error checking auth PIN %d: %v", pinID, err)
		http.Error(w, "Failed to verify authentication", http.StatusInternalServerError)
		return
	}

	// Check if we have an auth token
	if checkResp.AuthToken == "" {
		log.Printf("PIN %d not yet authenticated (no token)", pinID)
		http.Error(w, "Authentication not completed yet", http.StatusUnauthorized)
		return
	}

	log.Printf("PIN %d authenticated successfully, got token", pinID)

	// Verify the user has access to the server
	hasAccess, err := h.plexClient.CheckServerAccess(checkResp.AuthToken, h.config.PlexServerID)
	if err != nil {
		log.Printf("Error checking server access: %v", err)
		http.Error(w, "Failed to verify server access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		log.Println("User authenticated but does not have access to the server")
		http.Error(w, "You do not have access to this Plex server", http.StatusForbidden)
		return
	}

	// Create the session cookie
	cookie := &http.Cookie{
		Name:     "X-Plex-Token",
		Value:    checkResp.AuthToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	}

	if h.config.CookieDomain != "" {
		cookie.Domain = h.config.CookieDomain
	}

	http.SetCookie(w, cookie)

	log.Println("Authentication successful, session cookie created")

	// Return success status (for polling)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true, "message": "Authentication successful"}`))
}

// HandlePlexAuth shows an intermediate page that redirects to Plex
func (h *OAuthHandler) HandlePlexAuth(w http.ResponseWriter, r *http.Request) {
	authURL := r.URL.Query().Get("auth_url")
	if authURL == "" {
		http.Error(w, "Missing auth_url parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<title>Redirecting to Plex...</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			text-align: center;
			background-color: #1a1a1a;
			color: #fff;
		}
		h1 { color: #e5a00d; }
		.spinner {
			border: 3px solid #282828;
			border-top: 3px solid #e5a00d;
			border-radius: 50%%;
			width: 40px;
			height: 40px;
			animation: spin 1s linear infinite;
			margin: 20px auto;
		}
		@keyframes spin {
			0%% { transform: rotate(0deg); }
			100%% { transform: rotate(360deg); }
		}
	</style>
</head>
<body>
	<div class="spinner"></div>
	<h1>Redirecting to Plex...</h1>
	<p>Please wait while we redirect you to Plex for authentication.</p>
	<script>
		// Immediate redirect to Plex
		window.location.href = '%s';
	</script>
</body>
</html>
	`, authURL)))
}

// HandleClosePopup shows a page that closes the popup window
func (h *OAuthHandler) HandleClosePopup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
	<title>Authentication Complete</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			text-align: center;
			background-color: #1a1a1a;
			color: #fff;
		}
		h1 { color: #e5a00d; }
		.success-icon {
			font-size: 64px;
			margin: 20px 0;
		}
	</style>
</head>
<body>
	<div class="success-icon">✓</div>
	<h1>Authentication Complete!</h1>
	<p>This window will close automatically...</p>
	<script>
		// Try to close the window
		setTimeout(function() {
			window.close();
			// If window.close() doesn't work, show a message
			setTimeout(function() {
				document.body.innerHTML = '<h1>✓</h1><p>Authentication complete! You can close this window.</p>';
			}, 500);
		}, 1000);
	</script>
</body>
</html>
	`))
}

// HandleLogout clears the session cookie
func (h *OAuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get token before clearing to invalidate cache
	token := extractTokenFromRequest(r)
	if token != "" {
		h.tokenCache.Invalidate(token)
		log.Printf("Invalidated cached token on logout")
	}

	cookie := &http.Cookie{
		Name:     "X-Plex-Token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.CookieSecure,
		MaxAge:   -1, // Delete the cookie
	}

	if h.config.CookieDomain != "" {
		cookie.Domain = h.config.CookieDomain
	}

	http.SetCookie(w, cookie)

	log.Println("User logged out, session cookie cleared")

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Logged Out</title>
			<style>
				body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; text-align: center; }
				h1 { color: #e5a00d; }
			</style>
		</head>
		<body>
			<h1>Logged Out</h1>
			<p>You have been successfully logged out.</p>
			<p><a href="/login">Log in again</a></p>
		</body>
		</html>
	`))
}

// renderLoginPage renders the login page with Plex authentication
func (h *OAuthHandler) renderLoginPage(w http.ResponseWriter, authURL string, pinID int, code string, redirectURL string) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
	<title>Login with Plex</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			text-align: center;
			background-color: #1a1a1a;
			color: #fff;
		}
		h1 { color: #e5a00d; }
		p { color: #ccc; }
		.auth-button {
			display: inline-block;
			background-color: #e5a00d;
			color: #000;
			padding: 15px 40px;
			border: none;
			border-radius: 5px;
			font-weight: bold;
			font-size: 16px;
			margin: 20px 0;
			cursor: pointer;
			transition: background-color 0.2s;
		}
		.auth-button:hover {
			background-color: #cc8800;
		}
		.auth-button:disabled {
			background-color: #666;
			cursor: not-allowed;
			opacity: 0.6;
		}
		.pin-code {
			font-size: 24px;
			font-weight: bold;
			background-color: #282828;
			padding: 15px;
			border-radius: 5px;
			margin: 20px 0;
			letter-spacing: 4px;
		}
		.loading {
			margin-top: 30px;
			color: #999;
		}
		.spinner {
			border: 3px solid #282828;
			border-top: 3px solid #e5a00d;
			border-radius: 50%;
			width: 40px;
			height: 40px;
			animation: spin 1s linear infinite;
			margin: 20px auto;
		}
		@keyframes spin {
			0% { transform: rotate(0deg); }
			100% { transform: rotate(360deg); }
		}
	</style>
</head>
<body>
	<h1>Login with Plex</h1>
	<p>Authenticate with your Plex account to access this server.</p>
	<div class="pin-code">PIN: {{.Code}}</div>
	<button onclick="openAuthPopup()" class="auth-button" id="loginButton">
		Login with Plex
	</button>
	<div class="loading" id="loading" style="display:none;">
		<div class="spinner"></div>
		<p>Waiting for authentication...</p>
		<p style="font-size: 14px; margin-top: 10px;">Complete the authentication in the popup window.</p>
		<p style="font-size: 12px; color: #999; margin-top: 10px;">After approving, the popup will close automatically.</p>
	</div>
	<div id="status" style="margin-top: 20px;"></div>
	<script>
		let polling = false;
		let pollInterval;
		let authPopup;

		function openAuthPopup() {
			// Disable button and hide it
			const button = document.getElementById('loginButton');
			button.disabled = true;
			button.style.display = 'none';

			// Open popup window to our intermediate page
			const width = 600;
			const height = 700;
			const left = (screen.width - width) / 2;
			const top = (screen.height - height) / 2;

			// Open popup to our /auth/plex page instead of directly to Plex
			const plexAuthURL = '/auth/plex?auth_url=' + encodeURIComponent('{{.AuthURL}}');

			authPopup = window.open(
				plexAuthURL,
				'PlexAuth',
				'width=' + width + ',height=' + height + ',left=' + left + ',top=' + top + ',toolbar=no,menubar=no,scrollbars=yes,resizable=yes'
			);

			if (!authPopup) {
				// Popup blocked - show button again
				button.disabled = false;
				button.style.display = 'inline-block';
				document.getElementById('status').innerHTML =
					'<p style="color: #e5a00d;">Popup blocked! Please allow popups and try again.</p>' +
					'<p style="font-size: 14px; margin-top: 10px;">Or <a href="{{.AuthURL}}" target="_blank" style="color: #e5a00d;">click here</a> to open in a new tab.</p>';
				return;
			}

			// Start polling
			startPolling();

			// Show loading state
			document.getElementById('loading').style.display = 'block';

			// Check if popup is closed
			const popupChecker = setInterval(function() {
				if (authPopup && authPopup.closed) {
					clearInterval(popupChecker);
					if (polling) {
						// Give it a few more seconds to complete auth before giving up
						setTimeout(function() {
							if (polling) {
								stopPolling();
								document.getElementById('loading').style.display = 'none';
								document.getElementById('loginButton').style.display = 'inline-block';
								document.getElementById('loginButton').disabled = false;
								document.getElementById('status').innerHTML =
									'<p style="color: #e5a00d;">Authentication window closed before completing. Please try again.</p>';
							}
						}, 5000); // Give 5 seconds grace period
					}
				}
			}, 500);
		}

		function startPolling() {
			if (polling) return;
			polling = true;

			pollInterval = setInterval(checkAuth, 2000);
			// Stop polling after 5 minutes
			setTimeout(function() {
				stopPolling();
				document.getElementById('status').innerHTML =
					'<p style="color: #e5a00d;">Authentication timeout. <a href="#" onclick="openAuthPopup(); return false;" style="color: #e5a00d; text-decoration: underline;">Click here</a> to try again.</p>';
			}, 5 * 60 * 1000);
		}

		function stopPolling() {
			if (pollInterval) {
				clearInterval(pollInterval);
				polling = false;
			}
		}

		function checkAuth() {
			fetch('/callback?pin_id={{.PinID}}')
				.then(response => {
					if (response.ok) {
						stopPolling();
						// Close popup if still open
						if (authPopup && !authPopup.closed) {
							authPopup.close();
						}
						// Redirect to original URL
						window.location.href = '{{.RedirectURL}}';
					} else if (response.status === 403) {
						stopPolling();
						if (authPopup && !authPopup.closed) {
							authPopup.close();
						}
						document.getElementById('loading').style.display = 'none';
						document.getElementById('status').innerHTML =
							'<p style="color: #ff4444;">You do not have access to this Plex server.</p>';
					} else if (response.status !== 401) {
						// Some other error
						console.error('Auth check failed:', response.status);
					}
					// 401 means not authenticated yet, keep polling
				})
				.catch(error => {
					console.error('Error checking auth:', error);
				});
		}
	</script>
</body>
</html>
`

	t, err := template.New("login").Parse(tmpl)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"AuthURL":     authURL,
		"PinID":       pinID,
		"Code":        code,
		"RedirectURL": redirectURL,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		log.Printf("Error rendering login page: %v", err)
	}
}

// RenderSuccessPage renders the success page after authentication
func (h *OAuthHandler) RenderSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
	<title>Authentication Successful</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			text-align: center;
			background-color: #1a1a1a;
			color: #fff;
		}
		h1 { color: #e5a00d; }
		.success-icon {
			font-size: 64px;
			margin: 20px 0;
		}
		a {
			color: #e5a00d;
			text-decoration: none;
		}
		a:hover { text-decoration: underline; }
	</style>
</head>
<body>
	<div class="success-icon">✓</div>
	<h1>Authentication Successful!</h1>
	<p>You are now logged in and have access to the protected resources.</p>
	<p>You can close this window or <a href="/">return to home</a>.</p>
</body>
</html>
	`))
}

// CheckAuthStatus returns the authentication status as JSON
func (h *OAuthHandler) CheckAuthStatus(w http.ResponseWriter, r *http.Request) {
	token := extractTokenFromRequest(r)

	status := map[string]interface{}{
		"authenticated": false,
		"hasAccess":     false,
	}

	if token != "" {
		// Check cache first
		if cached, found := h.tokenCache.Get(token); found {
			status["authenticated"] = cached.Valid
			status["hasAccess"] = cached.HasAccess
			if cached.Username != "" {
				status["username"] = cached.Username
			}
		} else {
			// Cache miss - validate with Plex
			valid, _ := h.plexClient.ValidateToken(token)
			if valid {
				status["authenticated"] = true
				hasAccess, _ := h.plexClient.CheckServerAccess(token, h.config.PlexServerID)
				status["hasAccess"] = hasAccess

				// Get user info and cache the result
				userInfo, _ := h.plexClient.GetUserInfo(token)
				username := "Unknown"
				userID := 0
				if userInfo != nil {
					username = userInfo.Username
					userID = userInfo.ID
					status["username"] = username
				}

				// Cache the result
				h.tokenCache.Set(token, &cache.TokenCacheEntry{
					Valid:     true,
					HasAccess: hasAccess,
					UserID:    userID,
					Username:  username,
				})
			} else {
				// Cache invalid token
				h.tokenCache.Set(token, &cache.TokenCacheEntry{
					Valid:     false,
					HasAccess: false,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func extractTokenFromRequest(r *http.Request) string {
	// Try Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		if len(auth) > 7 && auth[:7] == "Bearer " {
			return auth[7:]
		}
		return auth
	}

	// Try X-Plex-Token header
	if token := r.Header.Get("X-Plex-Token"); token != "" {
		return token
	}

	// Try cookie
	if cookie, err := r.Cookie("X-Plex-Token"); err == nil {
		return cookie.Value
	}

	return ""
}
