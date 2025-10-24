package server

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// GeneratePinResponse represents the JSON response for PIN generation
type GeneratePinResponse struct {
	PinID   int    `json:"pin_id"`
	Code    string `json:"code"`
	AuthURL string `json:"auth_url"`
}

// GeneratePinHandler handles PIN generation requests
func (s *Server) GeneratePinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Request a PIN from Plex
	pinResp, err := s.PlexClient.CreateAuthPin()
	if err != nil {
		s.Logger.Error("Error requesting auth PIN", zap.Error(err))
		http.Error(w, "Failed to initiate authentication", http.StatusInternalServerError)
		return
	}

	s.Logger.Info("Generated auth PIN", zap.String("code", pinResp.Code), zap.Int("id", pinResp.ID))

	authURL := s.PlexClient.CreateAuthURL(pinResp.Code)

	// Return JSON response
	response := GeneratePinResponse{
		PinID:   pinResp.ID,
		Code:    pinResp.Code,
		AuthURL: authURL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.Logger.Error("Error encoding response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
