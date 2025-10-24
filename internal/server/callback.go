package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"
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

	s.Logger.Info("Checking PIN status", zap.Int("pin_id", pinID))
	checkResp, err := s.PlexClient.CheckAuthPin(pinID)
	if err != nil {
		s.Logger.Error("Error checking auth PIN", zap.Int("pin_id", pinID), zap.Error(err))
		http.Error(res, "Failed to verify authentication", http.StatusInternalServerError)
		return
	}

	if checkResp.AuthToken == "" {
		s.Logger.Info("PIN not yet authenticated (no token)", zap.Int("pin_id", pinID))
		http.Error(res, "Authentication not completed yet", http.StatusUnauthorized)
		return
	}

	s.Logger.Info("PIN authenticated successfully, got token", zap.Int("pin_id", pinID))

	user, err := s.PlexClient.GetUserInfo(checkResp.AuthToken)
	if err != nil || user == nil {
		s.Logger.Error("Error retrieving user info with token from PIN", zap.Int("pin_id", pinID), zap.Error(err))
		http.Error(res, "Failed to retrieve user info", http.StatusInternalServerError)
		return
	}

	// Verify the user has access to the server
	hasAccess, err := s.PlexClient.CheckServerAccess(checkResp.AuthToken)
	if err != nil {
		s.Logger.Error("Error checking server access", zap.Error(err))
		http.Error(res, "Failed to verify server access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		s.Logger.Info("User authenticated but does not have access to the server")
		http.Error(res, "You do not have access to this Plex server", http.StatusForbidden)
		return
	}

	s.createSessionCookie(res, checkResp.AuthToken)

	s.Logger.Info("Authentication successful, session cookie created")

	// Return success status (for polling)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(map[string]interface{}{
		"success": true,
		"message": "Authentication successful",
	})
}
