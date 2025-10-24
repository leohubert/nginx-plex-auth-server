package plex

import (
	"fmt"
)

// CheckAuthPinResponse represents the response when checking a PIN
type CheckAuthPinResponse struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	AuthToken string `json:"authToken"`
}

type apiCheckAuthPinResponse struct {
	CheckAuthPinResponse
	// Plex sometimes uses snake_case
	AuthTokenAlt string `json:"auth_token"`
}

// CheckAuthPin checks if a PIN has been authenticated
func (c *Client) CheckAuthPin(pinID int) (*CheckAuthPinResponse, error) {
	res, err := do[apiCheckAuthPinResponse](c.httpClient, &Request{
		Method: "GET",
		URL:    fmt.Sprintf("%s/api/v2/pins/%d", c.opts.BaseURL, pinID),
		Headers: map[string]string{
			"Accept":                   "application/json",
			"X-Plex-Client-Identifier": c.opts.ClientID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check auth pin: %w", err)
	}

	// Normalize auth token (Plex API might use authToken or auth_token)
	if res.AuthToken == "" && res.AuthTokenAlt != "" {
		res.AuthToken = res.AuthTokenAlt
	}

	return &CheckAuthPinResponse{
		ID:        res.ID,
		Code:      res.Code,
		AuthToken: res.AuthToken,
	}, nil
}
