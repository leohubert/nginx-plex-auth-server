package health

import (
	"log"
	"sync"
	"time"

	"github.com/hubert_i/nginx_plex_auth_server/pkg/plex"
)

// TokenStatus represents the health status of the owner token
type TokenStatus struct {
	Valid         bool      `json:"valid"`
	LastChecked   time.Time `json:"last_checked"`
	LastError     string    `json:"last_error,omitempty"`
	OwnerUsername string    `json:"owner_username,omitempty"`
	OwnerID       int       `json:"owner_id,omitempty"`
}

// TokenMonitor periodically checks the health of the Plex owner token
type TokenMonitor struct {
	plexClient     *plex.Client
	ownerToken     string
	checkInterval  time.Duration
	status         TokenStatus
	statusMu       sync.RWMutex
	stopChan       chan struct{}
	onInvalidToken func(error)
}

// NewTokenMonitor creates a new token health monitor
func NewTokenMonitor(client *plex.Client, ownerToken string, checkInterval time.Duration) *TokenMonitor {
	return &TokenMonitor{
		plexClient:    client,
		ownerToken:    ownerToken,
		checkInterval: checkInterval,
		status: TokenStatus{
			Valid:       false,
			LastChecked: time.Time{},
		},
		stopChan: make(chan struct{}),
	}
}

// SetInvalidTokenCallback sets a callback function that will be called when token becomes invalid
func (m *TokenMonitor) SetInvalidTokenCallback(callback func(error)) {
	m.onInvalidToken = callback
}

// Start begins the periodic token health checks
func (m *TokenMonitor) Start() {
	log.Printf("Starting token health monitor (check interval: %v)", m.checkInterval)

	// Do an immediate check on startup
	m.check()

	// Start periodic checks
	ticker := time.NewTicker(m.checkInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.check()
			case <-m.stopChan:
				ticker.Stop()
				log.Println("Token health monitor stopped")
				return
			}
		}
	}()
}

// Stop stops the periodic health checks
func (m *TokenMonitor) Stop() {
	close(m.stopChan)
}

// check validates the owner token and updates the status
func (m *TokenMonitor) check() {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	m.status.LastChecked = time.Now()

	// Validate the token
	valid, err := m.plexClient.ValidateToken(m.ownerToken)
	if err != nil {
		m.status.Valid = false
		m.status.LastError = err.Error()
		log.Printf("⚠️  Token health check failed: %v", err)

		// Call the callback if token validation failed
		if m.onInvalidToken != nil {
			m.onInvalidToken(err)
		}
		return
	}

	if !valid {
		m.status.Valid = false
		m.status.LastError = "Token is invalid or expired"
		log.Printf("❌ CRITICAL: Owner token is INVALID. Please update PLEX_TOKEN environment variable!")

		// Call the callback if token is invalid
		if m.onInvalidToken != nil {
			m.onInvalidToken(nil)
		}
		return
	}

	// Token is valid - get owner info
	userInfo, err := m.plexClient.GetUserInfo(m.ownerToken)
	if err != nil {
		// Token is valid but couldn't get user info
		m.status.Valid = true
		m.status.LastError = "Could not fetch owner info: " + err.Error()
		log.Printf("⚠️  Token is valid but could not fetch owner info: %v", err)
		return
	}

	// Update status with success
	previousValid := m.status.Valid
	m.status.Valid = true
	m.status.LastError = ""
	m.status.OwnerUsername = userInfo.Username
	m.status.OwnerID = userInfo.ID

	// Log only if status changed or this is the first check
	if !previousValid || m.status.OwnerID == 0 {
		log.Printf("✓ Token health check passed (Owner: %s, ID: %d)", userInfo.Username, userInfo.ID)
	}
}

// GetStatus returns the current token health status
func (m *TokenMonitor) GetStatus() TokenStatus {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()

	// Return a copy to prevent race conditions
	return TokenStatus{
		Valid:         m.status.Valid,
		LastChecked:   m.status.LastChecked,
		LastError:     m.status.LastError,
		OwnerUsername: m.status.OwnerUsername,
		OwnerID:       m.status.OwnerID,
	}
}

// IsHealthy returns true if the token is currently valid
func (m *TokenMonitor) IsHealthy() bool {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	return m.status.Valid
}