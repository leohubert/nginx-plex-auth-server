package plex

import (
	"fmt"
)

type Connection struct {
	Protocol string `xml:"protocol,attr"`
	Address  string `xml:"address,attr"`
	Port     string `xml:"port,attr"`
	URI      string `xml:"uri,attr"`
	Local    string `xml:"local,attr,omitempty"`
	Relay    string `xml:"relay,attr,omitempty"`
}

type Device struct {
	Name             string `xml:"name,attr"`
	Product          string `xml:"product,attr"`
	ProductVersion   string `xml:"productVersion,attr"`
	Platform         string `xml:"platform,attr"`
	PlatformVersion  string `xml:"platformVersion,attr"`
	Device           string `xml:"device,attr"`
	ClientIdentifier string `xml:"clientIdentifier,attr"`
}

type MediaContainer struct {
	Size    string   `xml:"size,attr"`
	Devices []Device `xml:"Device"`
}

// CheckServerAccess validates if a user has access to a specific Plex server
func (c *Client) CheckServerAccess(userToken string) (bool, error) {
	// Next, get the list of servers shared with the user
	apiResources, err := do[MediaContainer](c.httpClient, &Request{
		Method: "GET",
		URL:    c.opts.BaseURL + "/api/resources?includeHttps=1&includeRelay=1&includeSharedServers=1",
		Headers: map[string]string{
			"X-Plex-Token":             userToken,
			"X-Plex-Client-Identifier": c.opts.ClientID,
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to get shared servers: %w", err)
	}

	// Check if the specified server ID is in the list of shared servers
	hasAccess := false
	for _, device := range apiResources.Devices {
		if device.ClientIdentifier == c.opts.ServerID {
			hasAccess = true
			break
		}
	}

	return hasAccess, nil
}
