package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hubert_i/nginx_plex_auth_server/internal/auth"
	"github.com/hubert_i/nginx_plex_auth_server/internal/config"
	"github.com/hubert_i/nginx_plex_auth_server/internal/health"
	"github.com/hubert_i/nginx_plex_auth_server/pkg/plex"
)

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

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create Plex client
	plexClient := plex.NewClient(cfg.PlexURL, cfg.PlexToken, cfg.PlexClientID)

	// Validate Plex token at startup
	log.Println("Validating Plex token...")
	valid, err := plexClient.ValidateToken(cfg.PlexToken)
	if err != nil {
		log.Fatalf("Failed to validate Plex token: %v", err)
	}
	if !valid {
		log.Fatalf("Plex token is invalid. Please check your PLEX_TOKEN environment variable.")
	}
	log.Println("✓ Plex token validated successfully")

	// Verify server access
	log.Printf("Verifying access to Plex server: %s", cfg.PlexServerID)
	userInfo, err := plexClient.GetUserInfo(cfg.PlexToken)
	if err != nil {
		log.Fatalf("Failed to get user info from Plex token: %v", err)
	}
	log.Printf("✓ Authenticated as: %s (ID: %d)", userInfo.Username, userInfo.ID)

	// Initialize token health monitor
	tokenMonitor := health.NewTokenMonitor(plexClient, cfg.PlexToken, cfg.TokenHealthCheckTTL)

	// Set callback for when token becomes invalid
	tokenMonitor.SetInvalidTokenCallback(func(err error) {
		if err != nil {
			log.Printf("⚠️  ALERT: Token validation failed: %v", err)
		} else {
			log.Printf("❌ CRITICAL ALERT: PLEX_TOKEN is INVALID! Update your environment variable immediately!")
		}
	})

	// Start the monitor
	tokenMonitor.Start()
	defer tokenMonitor.Stop()

	// Create handlers
	authHandler := auth.NewHandler(cfg)
	oauthHandler := auth.NewOAuthHandler(cfg, plexClient)
	healthHandler := health.NewHandler(tokenMonitor)

	// Setup routes
	// Auth endpoint for Nginx auth_request (returns status codes only)
	http.HandleFunc("/auth", authHandler.HandleAuth)

	// OAuth flow endpoints
	http.HandleFunc("/login", oauthHandler.HandleLogin)
	http.HandleFunc("/auth/plex", oauthHandler.HandlePlexAuth)
	http.HandleFunc("/callback", oauthHandler.HandleCallback)
	http.HandleFunc("/logout", oauthHandler.HandleLogout)

	// Status endpoint
	http.HandleFunc("/status", oauthHandler.CheckAuthStatus)

	// Health check endpoints
	http.HandleFunc("/health", healthHandler.HandleHealthCheck)
	http.HandleFunc("/health/token", healthHandler.HandleTokenHealth)
	http.HandleFunc("/health/detailed", healthHandler.HandleDetailedHealth)

	// Root endpoint - show welcome page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact root path
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		token := extractTokenFromRequest(r)
		if token == "" {
			// Not logged in - show login prompt
							w.Header().Set("Content-Type", "text/html; charset=utf-8")

			w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
	<title>Nginx Plex Auth Server</title>
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
		a {
			display: inline-block;
			background-color: #e5a00d;
			color: #000;
			padding: 15px 40px;
			border: none;
			border-radius: 5px;
			font-weight: bold;
			font-size: 16px;
			margin: 20px 0;
			text-decoration: none;
			transition: background-color 0.2s;
		}
		a:hover { background-color: #cc8800; }
	</style>
</head>
<body>
	<h1>Nginx Plex Auth Server</h1>
	<p>Authentication server for Nginx using Plex OAuth</p>
	<a href="/login">Login with Plex</a>
	<p style="margin-top: 40px; font-size: 14px;">
		<a href="/status" style="font-size: 14px; padding: 10px 20px; background-color: #282828;">Check Status</a>
	</p>
</body>
</html>
			`))
		} else {
			// Logged in - show status
			valid, _ := plexClient.ValidateToken(token)
			if valid {
				hasAccess, _ := plexClient.CheckServerAccess(token, cfg.PlexServerID)
				userInfo, _ := plexClient.GetUserInfo(token)

				username := "Unknown"
				if userInfo != nil {
					username = userInfo.Username
				}

				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<title>Nginx Plex Auth Server</title>
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
		.status {
			background-color: #282828;
			padding: 20px;
			border-radius: 5px;
			margin: 20px 0;
		}
		.status-ok { color: #4CAF50; }
		.status-error { color: #f44336; }
		a {
			display: inline-block;
			background-color: #e5a00d;
			color: #000;
			padding: 10px 30px;
			border: none;
			border-radius: 5px;
			font-weight: bold;
			font-size: 14px;
			margin: 10px 5px;
			text-decoration: none;
			transition: background-color 0.2s;
		}
		a:hover { background-color: #cc8800; }
	</style>
</head>
<body>
	<h1>Nginx Plex Auth Server</h1>
	<div class="status">
		<p><strong>Logged in as:</strong> %s</p>
		<p class="%s"><strong>Server Access:</strong> %s</p>
	</div>
	<a href="/status">Check Status (JSON)</a>
	<a href="/logout" style="background-color: #666;">Logout</a>
</body>
</html>
				`, username,
				   func() string { if hasAccess { return "status-ok" } else { return "status-error" } }(),
				   func() string { if hasAccess { return "Granted ✓" } else { return "Denied ✗" } }())))
			} else {
				// Invalid token

			}
		}
	})

	// Start server
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Starting Nginx auth server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
