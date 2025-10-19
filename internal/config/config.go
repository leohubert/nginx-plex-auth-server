package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	PlexURL              string
	PlexToken            string
	PlexServerID         string
	PlexClientID         string
	ServerAddr           string
	CallbackURL          string
	CookieDomain         string
	CookieSecure         bool
	CacheTTL             time.Duration
	CacheMaxSize         int
	TokenHealthCheckTTL  time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		PlexURL:      os.Getenv("PLEX_URL"),
		PlexToken:    os.Getenv("PLEX_TOKEN"),
		PlexServerID: os.Getenv("PLEX_SERVER_ID"),
		PlexClientID: os.Getenv("PLEX_CLIENT_ID"),
		ServerAddr:   os.Getenv("SERVER_ADDR"),
		CallbackURL:  os.Getenv("CALLBACK_URL"),
		CookieDomain: os.Getenv("COOKIE_DOMAIN"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
	}

	// Set defaults
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = ":8080"
	}

	if cfg.PlexURL == "" {
		cfg.PlexURL = "https://plex.tv"
	}

	if cfg.CallbackURL == "" {
		cfg.CallbackURL = "http://localhost:8080/callback"
	}

	if cfg.PlexClientID == "" {
		cfg.PlexClientID = "plex-auth-nginx-module"
	}

	// Cache configuration with defaults
	cacheTTLSeconds := 300 // Default 5 minutes
	if ttlEnv := os.Getenv("CACHE_TTL_SECONDS"); ttlEnv != "" {
		if ttl, err := strconv.Atoi(ttlEnv); err == nil && ttl > 0 {
			cacheTTLSeconds = ttl
		}
	}
	cfg.CacheTTL = time.Duration(cacheTTLSeconds) * time.Second

	cfg.CacheMaxSize = 1000 // Default max 1000 tokens
	if maxSizeEnv := os.Getenv("CACHE_MAX_SIZE"); maxSizeEnv != "" {
		if maxSize, err := strconv.Atoi(maxSizeEnv); err == nil && maxSize > 0 {
			cfg.CacheMaxSize = maxSize
		}
	}

	// Token health check configuration
	tokenHealthCheckSeconds := 300 // Default 5 minutes
	if healthEnv := os.Getenv("TOKEN_HEALTH_CHECK_INTERVAL"); healthEnv != "" {
		if interval, err := strconv.Atoi(healthEnv); err == nil && interval > 0 {
			tokenHealthCheckSeconds = interval
		}
	}
	cfg.TokenHealthCheckTTL = time.Duration(tokenHealthCheckSeconds) * time.Second

	// Validate required fields
	if cfg.PlexToken == "" {
		return nil, fmt.Errorf("PLEX_TOKEN environment variable is required")
	}

	if cfg.PlexServerID == "" {
		return nil, fmt.Errorf("PLEX_SERVER_ID environment variable is required")
	}

	return cfg, nil
}
