package auth

import (
	"log"
	"net/http"

	"github.com/hubert_i/nginx_plex_auth_server/internal/cache"
	"github.com/hubert_i/nginx_plex_auth_server/internal/config"
	"github.com/hubert_i/nginx_plex_auth_server/pkg/plex"
)

// Handler manages authentication requests
type Handler struct {
	config      *config.Config
	plexClient  *plex.Client
	tokenCache  *cache.TokenCache
}

// NewHandler creates a new authentication handler
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		config:     cfg,
		plexClient: plex.NewClient(cfg.PlexURL, cfg.PlexToken, cfg.PlexClientID),
		tokenCache: cache.NewTokenCache(cfg.CacheTTL, cfg.CacheMaxSize),
	}
}

// HandleAuth processes Nginx auth_request subrequests
func (h *Handler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	// Extract authentication token from header or cookie
	token := h.extractToken(r)

	if token == "" {
		log.Println("No authentication token provided")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check cache first
	if cached, found := h.tokenCache.Get(token); found {
		log.Println("Using cached token validation result")
		if !cached.Valid {
			log.Println("Invalid authentication token (cached)")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !cached.HasAccess {
			log.Println("User does not have access to the specified Plex server (cached)")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		log.Printf("Authentication and server access validation successful (cached, user: %s)", cached.Username)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Cache miss - validate with Plex
	log.Println("Cache miss - validating token with Plex")
	valid, err := h.plexClient.ValidateToken(token)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !valid {
		// Cache the invalid result
		h.tokenCache.Set(token, &cache.TokenCacheEntry{
			Valid:     false,
			HasAccess: false,
		})
		log.Println("Invalid authentication token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check if user has access to the specified Plex server
	hasAccess, err := h.plexClient.CheckServerAccess(token, h.config.PlexServerID)
	if err != nil {
		log.Printf("Error checking server access: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get user info for caching
	userInfo, _ := h.plexClient.GetUserInfo(token)
	username := "Unknown"
	userID := 0
	if userInfo != nil {
		username = userInfo.Username
		userID = userInfo.ID
	}

	// Cache the result
	h.tokenCache.Set(token, &cache.TokenCacheEntry{
		Valid:     true,
		HasAccess: hasAccess,
		UserID:    userID,
		Username:  username,
	})

	if !hasAccess {
		log.Println("User does not have access to the specified Plex server")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Authentication and authorization successful
	log.Printf("Authentication and server access validation successful (user: %s)", username)
	w.WriteHeader(http.StatusOK)
}

// extractToken retrieves the authentication token from the request
func (h *Handler) extractToken(r *http.Request) string {
	// Try Authorization header first
	if auth := r.Header.Get("Authorization"); auth != "" {
		// Support "Bearer <token>" format
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

// HandleAuthWithRedirect processes auth requests and redirects browsers to login
// Use this for user-facing endpoints that should redirect to login page
func (h *Handler) HandleAuthWithRedirect(w http.ResponseWriter, r *http.Request) {
	// Extract authentication token from header or cookie
	token := h.extractToken(r)

	if token == "" {
		log.Println("No authentication token provided, redirecting to login")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Validate token with Plex
	valid, err := h.plexClient.ValidateToken(token)
	if err != nil {
		log.Printf("Error validating token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !valid {
		log.Println("Invalid authentication token, redirecting to login")
		// Clear the invalid cookie
		http.SetCookie(w, &http.Cookie{
			Name:   "X-Plex-Token",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Check if user has access to the specified Plex server
	hasAccess, err := h.plexClient.CheckServerAccess(token, h.config.PlexServerID)
	if err != nil {
		log.Printf("Error checking server access: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		log.Println("User does not have access to the specified Plex server")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("You do not have access to this server"))
		return
	}

	// Authentication and authorization successful
	log.Println("Authentication and server access validation successful")
	w.WriteHeader(http.StatusOK)
}
