package server

import (
	"encoding/json"
	"log"
	"net/http"
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
		log.Printf("Error requesting auth PIN: %v", err)
		http.Error(w, "Failed to initiate authentication", http.StatusInternalServerError)
		return
	}

	log.Printf("Generated auth PIN: %s (ID: %d)", pinResp.Code, pinResp.ID)

	authURL := s.PlexClient.CreateAuthURL(pinResp.Code)

	// Return JSON response
	response := GeneratePinResponse{
		PinID:   pinResp.ID,
		Code:    pinResp.Code,
		AuthURL: authURL,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
