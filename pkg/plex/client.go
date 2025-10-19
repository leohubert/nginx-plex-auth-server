package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents a Plex API client
type Client struct {
	baseURL    string
	token      string
	clientID   string
	httpClient *http.Client
}

// NewClient creates a new Plex API client
func NewClient(baseURL, token, clientID string) *Client {
	return &Client{
		baseURL:  baseURL,
		token:    token,
		clientID: clientID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateToken checks if a Plex token is valid
func (c *Client) ValidateToken(token string) (bool, error) {
	// Use the identity endpoint which returns JSON
	req, err := http.NewRequest("GET", c.baseURL+"/api/v2/user", nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Use the provided token for validation
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// UserInfo represents basic Plex user information
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ServerAccessResponse represents the response from server shared users endpoint
type ServerAccessResponse struct {
	MediaContainer struct {
		User []struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"User"`
	} `json:"MediaContainer"`
}

// CheckServerAccess validates if a user has access to a specific Plex server
func (c *Client) CheckServerAccess(userToken, serverID string) (bool, error) {
	// First, get the user info from their token
	userInfo, err := c.GetUserInfo(userToken)
	if err != nil {
		return false, fmt.Errorf("failed to get user info: %w", err)
	}

	// Check if this is the server owner
	ownerInfo, err := c.GetUserInfo(c.token)
	if err != nil {
		return false, fmt.Errorf("failed to get owner info: %w", err)
	}

	// If the user is the owner, they have access
	if userInfo.ID == ownerInfo.ID {
		return true, nil
	}

	// Check if user has access via shared servers
	hasAccess, err := c.checkSharedServerAccess(userInfo.ID, serverID)
	if err != nil {
		return false, fmt.Errorf("failed to check shared access: %w", err)
	}

	return hasAccess, nil
}

// GetUserInfo retrieves user information from a token
func (c *Client) GetUserInfo(token string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v2/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userInfo, nil
}

// checkSharedServerAccess checks if a user has access to a shared server
func (c *Client) checkSharedServerAccess(userID int, serverID string) (bool, error) {
	// Get list of users with access to the server
	url := fmt.Sprintf("%s/api/v2/shared_servers/%s", c.baseURL, serverID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accessResp ServerAccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&accessResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if the user ID is in the list of users with access
	for _, user := range accessResp.MediaContainer.User {
		if user.ID == userID {
			return true, nil
		}
	}

	return false, nil
}

// AuthPinResponse represents the response when requesting a PIN
type AuthPinResponse struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

// AuthPinCheckResponse represents the response when checking a PIN
type AuthPinCheckResponse struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	AuthToken string `json:"authToken"`
	// Plex sometimes uses snake_case
	AuthTokenAlt string `json:"auth_token"`
}

// RequestAuthPin requests a new authentication PIN from Plex
func (c *Client) RequestAuthPin() (*AuthPinResponse, error) {
	// Add strong=true parameter as per Overseerr implementation
	req, err := http.NewRequest("POST", c.baseURL+"/api/v2/pins?strong=true", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set all required Plex headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Product", "Nginx Auth Server")
	req.Header.Set("X-Plex-Version", "1.0")
	req.Header.Set("X-Plex-Client-Identifier", c.clientID)
	req.Header.Set("X-Plex-Model", "Plex OAuth")
	req.Header.Set("X-Plex-Platform", "Web")
	req.Header.Set("X-Plex-Platform-Version", "1.0")
	req.Header.Set("X-Plex-Device", "Linux")
	req.Header.Set("X-Plex-Device-Name", "Nginx Auth Server")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var pinResp AuthPinResponse
	if err := json.NewDecoder(resp.Body).Decode(&pinResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pinResp, nil
}

// CheckAuthPin checks if a PIN has been authenticated
func (c *Client) CheckAuthPin(pinID int) (*AuthPinCheckResponse, error) {
	url := fmt.Sprintf("%s/api/v2/pins/%d", c.baseURL, pinID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var checkResp AuthPinCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Normalize auth token (Plex API might use authToken or auth_token)
	if checkResp.AuthToken == "" && checkResp.AuthTokenAlt != "" {
		checkResp.AuthToken = checkResp.AuthTokenAlt
	}

	return &checkResp, nil
}
