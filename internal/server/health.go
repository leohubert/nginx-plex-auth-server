package server

import (
	"net/http"
)

// HealthHandler initiates the Plex OAuth flow
func (s *Server) HealthHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
}
