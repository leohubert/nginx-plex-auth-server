package main

import (
	"log"
	"net/http"
	"os"

	"github.com/hubert_i/nginx_plex_auth_server/internal/auth"
	"github.com/hubert_i/nginx_plex_auth_server/internal/config"
	"github.com/hubert_i/nginx_plex_auth_server/pkg/plex"
)

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

	// Create handlers
	authHandler := auth.NewHandler(cfg)
	oauthHandler := auth.NewOAuthHandler(cfg, plexClient)

	// Setup routes
	// Auth endpoint for Nginx auth_request (returns status codes only)
	http.HandleFunc("/auth", authHandler.HandleAuth)

	// OAuth flow endpoints
	http.HandleFunc("/login", oauthHandler.HandleLogin)
	http.HandleFunc("/auth/plex", oauthHandler.HandlePlexAuth)
	http.HandleFunc("/callback", oauthHandler.HandleCallback)
	http.HandleFunc("/close-popup", oauthHandler.HandleClosePopup)
	http.HandleFunc("/auth-success", func(w http.ResponseWriter, r *http.Request) {
		oauthHandler.RenderSuccessPage(w)
	})
	http.HandleFunc("/logout", oauthHandler.HandleLogout)

	// Status endpoint
	http.HandleFunc("/status", oauthHandler.CheckAuthStatus)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
