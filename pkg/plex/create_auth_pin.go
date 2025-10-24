package plex

import "fmt"

// AuthPinResponse represents the response when requesting a PIN
type AuthPinResponse struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

// CreateAuthPin requests a new authentication PIN from Plex
func (c *Client) CreateAuthPin() (*AuthPinResponse, error) {

	pinResp, err := do[AuthPinResponse](c.httpClient, &Request{
		Method: "POST",
		URL:    c.opts.BaseURL + "/api/v2/pins?strong=true",
		Headers: map[string]string{
			"Accept":                   "application/json",
			"X-Plex-Product":           "Nginx Auth Server",
			"X-Plex-Version":           "1.0",
			"X-Plex-Client-Identifier": c.opts.ClientID,
			"X-Plex-Model":             "Plex OAuth",
			"X-Plex-Platform":          "Web",
			"X-Plex-Platform-Version":  "1.0",
			"X-Plex-Devices":           "Linux",
			"X-Plex-Devices-Name":      "Nginx Auth Server",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to request auth PIN: %w", err)
	}

	return pinResp, nil
}
