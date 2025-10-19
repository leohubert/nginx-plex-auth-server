package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler manages health check endpoints
type Handler struct {
	tokenMonitor *TokenMonitor
	startTime    time.Time
}

// NewHandler creates a new health check handler
func NewHandler(tokenMonitor *TokenMonitor) *Handler {
	return &Handler{
		tokenMonitor: tokenMonitor,
		startTime:    time.Now(),
	}
}

// HandleHealthCheck returns basic health status
func (h *Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Basic health check - returns OK if service is running
	// This is used by container orchestration systems
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// HandleTokenHealth returns detailed token health status
func (h *Handler) HandleTokenHealth(w http.ResponseWriter, r *http.Request) {
	status := h.tokenMonitor.GetStatus()

	// Determine HTTP status code based on token validity
	httpStatus := http.StatusOK
	if !status.Valid {
		// Token is invalid - service is degraded
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(status)
}

// HandleDetailedHealth returns comprehensive health information
func (h *Handler) HandleDetailedHealth(w http.ResponseWriter, r *http.Request) {
	tokenStatus := h.tokenMonitor.GetStatus()

	response := map[string]interface{}{
		"status":  "healthy",
		"uptime":  time.Since(h.startTime).String(),
		"token":   tokenStatus,
		"service": "nginx-plex-auth-server",
	}

	// If token is invalid, mark overall status as degraded
	if !tokenStatus.Valid {
		response["status"] = "degraded"
		response["message"] = "Owner token is invalid - authentication will fail"
	}

	httpStatus := http.StatusOK
	if !tokenStatus.Valid {
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}