package health

import (
	"context"
	"encoding/json"
	"net/http"
	"shared/server/response"
	"time"
)

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	healthResponse := h.manager.Check(ctx)

	statusCode := response.StatusOK
	if healthResponse.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}

func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	healthResponse := h.manager.Check(ctx)

	statusCode := response.StatusOK
	if healthResponse.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}
