package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// CallbackHandler handles polling requests to check PIN authentication status
func (s *Server) CallbackHandler(res http.ResponseWriter, req *http.Request) {
	// Get the PIN ID from query params
	pinIDStr := req.URL.Query().Get("pin_id")
	if pinIDStr == "" {
		http.Error(res, "Missing pin_id parameter", http.StatusBadRequest)
		return
	}

	pinID, err := strconv.Atoi(pinIDStr)
	if err != nil {
		http.Error(res, "Invalid pin_id parameter", http.StatusBadRequest)
		return
	}

	// Check the PIN status
	log.Printf("Checking PIN %d status...", pinID)
	checkResp, err := s.PlexClient.CheckAuthPin(pinID)
	if err != nil {
		log.Printf("Error checking auth PIN %d: %v", pinID, err)
		http.Error(res, "Failed to verify authentication", http.StatusInternalServerError)
		return
	}

	if checkResp.AuthToken == "" {
		log.Printf("PIN %d not yet authenticated (no token)", pinID)
		http.Error(res, "Authentication not completed yet", http.StatusUnauthorized)
		return
	}

	log.Printf("PIN %d authenticated successfully, got token", pinID)

	user, err := s.PlexClient.GetUserInfo(checkResp.AuthToken)
	if err != nil || user == nil {
		log.Printf("Error retrieving user info with token from PIN %d: %v", pinID, err)
		http.Error(res, "Failed to retrieve user info", http.StatusInternalServerError)
		return
	}

	// Verify the user has access to the server
	hasAccess, err := s.PlexClient.CheckServerAccess(checkResp.AuthToken)
	if err != nil {
		log.Printf("Error checking server access: %v", err)
		http.Error(res, "Failed to verify server access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		log.Println("User authenticated but does not have access to the server")
		http.Error(res, "You do not have access to this Plex server", http.StatusForbidden)
		return
	}

	s.createSessionCookie(res, checkResp.AuthToken)

	log.Println("Authentication successful, session cookie created")

	// Return success status (for polling)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(map[string]interface{}{
		"success": true,
		"message": "Authentication successful",
	})
}
