package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	PlexURL         string
	PlexToken       string
	PlexServerID    string
	PlexClientID    string
	ServerAddr      string
	CallbackURL     string
	CookieDomain    string
	CookieSecure    bool
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
		cfg.PlexClientID = "cd72c25b-4d05-41d1-8aec-66a907585452"
	}

	// Validate required fields
	if cfg.PlexToken == "" {
		return nil, fmt.Errorf("PLEX_TOKEN environment variable is required")
	}

	if cfg.PlexServerID == "" {
		return nil, fmt.Errorf("PLEX_SERVER_ID environment variable is required")
	}

	return cfg, nil
}
