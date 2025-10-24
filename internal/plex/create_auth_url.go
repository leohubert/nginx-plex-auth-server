package plex

import "fmt"

func (c *Client) CreateAuthURL(pinCode string) string {
	return fmt.Sprintf("%s/auth/#!?clientID=%s&context[device][product]=%s&context[device][version]=%s&context[device][platform]=%s&context[device][platformVersion]=%s&context[device][device]=%s&context[device][deviceName]=%s&context[device][model]=%s&context[device][layout]=%s&code=%s",
		"https://app.plex.tv",
		c.ClientID,
		"Nginx+Auth+Server",
		"1.0",
		"Web",
		"1.0",
		"Linux",
		"Nginx+Auth+Server",
		"Plex+OAuth",
		"desktop",
		pinCode,
	)
}
