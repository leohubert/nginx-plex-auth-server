package cache

import (
	"sync"
	"time"
)

// TokenCacheEntry represents a cached token validation result
type TokenCacheEntry struct {
	Valid      bool
	HasAccess  bool
	UserID     int
	Username   string
	ExpiresAt  time.Time
}

// TokenCache provides a thread-safe cache for token validation results
type TokenCache struct {
	mu      sync.RWMutex
	entries map[string]*TokenCacheEntry
	ttl     time.Duration
	maxSize int
}

// NewTokenCache creates a new token cache with specified TTL and max size
func NewTokenCache(ttl time.Duration, maxSize int) *TokenCache {
	cache := &TokenCache{
		entries: make(map[string]*TokenCacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached token validation result
func (c *TokenCache) Get(token string) (*TokenCacheEntry, bool) {
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
func (c *TokenCache) Set(token string, entry *TokenCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	entry.ExpiresAt = time.Now().Add(c.ttl)
	c.entries[token] = entry
}

// Invalidate removes a token from the cache
func (c *TokenCache) Invalidate(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, token)
}

// Clear removes all entries from the cache
func (c *TokenCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*TokenCacheEntry)
}

// Size returns the current number of cached entries
func (c *TokenCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictOldest removes the oldest entry from the cache
// Must be called with lock held
func (c *TokenCache) evictOldest() {
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
func (c *TokenCache) cleanupExpired() {
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