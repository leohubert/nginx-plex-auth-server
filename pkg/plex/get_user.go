package plex

import "fmt"

// UserInfo represents basic Plex user information
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// GetUserInfo retrieves user information from a token
func (c *Client) GetUserInfo(token string) (*UserInfo, error) {
	userInfo, err := do[UserInfo](c.httpClient, &Request{
		Method: "GET",
		URL:    c.opts.BaseURL + "/api/v2/user",
		Headers: map[string]string{
			"X-Plex-Token": token,
			"Accept":       "application/json",
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return userInfo, nil
}
