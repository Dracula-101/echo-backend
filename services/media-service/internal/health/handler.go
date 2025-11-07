package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Handler handles health check HTTP requests
type Handler struct {
	manager *Manager
}

// NewHandler creates a new health handler
func NewHandler(manager *Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// Health returns the full health status
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := h.manager.CheckHealth(ctx)

	statusCode := http.StatusOK
	if response.Status == string(StatusDegraded) {
		statusCode = http.StatusOK // Still return 200 for degraded
	} else if response.Status == string(StatusUnhealthy) {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Liveness returns a simple liveness check (service is running)
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// Readiness returns readiness status (service is ready to handle requests)
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := h.manager.CheckHealth(ctx)

	statusCode := http.StatusOK
	if response.Status == string(StatusUnhealthy) {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"status": response.Status,
	})
}
