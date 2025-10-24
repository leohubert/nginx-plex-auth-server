package cache

import (
	"sync"
	"time"
)

// TokenCacheEntry represents a cached token validation result
type TokenCacheEntry struct {
	Valid     bool
	HasAccess bool
	ExpiresAt time.Time
}

// CacheClient provides a thread-safe cache for token validation results

type Options struct {
	TTL     time.Duration
	MaxSize int
}

type Client struct {
	opts    Options
	mu      sync.RWMutex
	entries map[string]*TokenCacheEntry
}

// NewCacheClient creates a new token cache with specified TTL and max size
func NewCacheClient(opts Options) *Client {
	cache := &Client{
		opts:    opts,
		entries: make(map[string]*TokenCacheEntry),
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached token validation result
func (c *Client) Get(token string) (*TokenCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[token]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry, true
}

// Set stores a token validation result in the cache
func (c *Client) Set(token string, entry *TokenCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if len(c.entries) >= c.opts.MaxSize {
		c.evictOldest()
	}

	entry.ExpiresAt = time.Now().Add(c.opts.TTL)
	c.entries[token] = entry
}

// Invalidate removes a token from the cache
func (c *Client) Invalidate(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, token)
}

// Clear removes all entries from the cache
func (c *Client) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*TokenCacheEntry)
}

// Size returns the current number of cached entries
func (c *Client) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictOldest removes the oldest entry from the cache
// Must be called with lock held
func (c *Client) evictOldest() {
	var oldestToken string
	var oldestTime time.Time

	for token, entry := range c.entries {
		if oldestToken == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestToken = token
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestToken != "" {
		delete(c.entries, oldestToken)
	}
}

// cleanupExpired periodically removes expired entries
func (c *Client) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for token, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				delete(c.entries, token)
			}
		}
		c.mu.Unlock()
	}
}
