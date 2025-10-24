package plex

import (
	"net/http"
	"time"
)

type Options struct {
	BaseURL  string
	ClientID string
	ServerID string
}

// Client represents a Plex API client
type Client struct {
	opts       Options
	httpClient *http.Client
}

// NewClient creates a new Plex API client
func NewClient(opts Options) *Client {
	return &Client{
		opts: opts,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
