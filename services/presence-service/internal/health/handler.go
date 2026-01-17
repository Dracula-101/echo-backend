package health

import (
	"encoding/json"
	"net/http"
	"shared/server/response"
)

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := h.manager.Check(ctx)

	w.Header().Set("Content-Type", "application/json")

	status := result["status"]
	if status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(response.StatusOK)
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := h.manager.Liveness(ctx)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := h.manager.Readiness(ctx)

	w.Header().Set("Content-Type", "application/json")

	status := result["status"]
	if status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(response.StatusOK)
	}

	json.NewEncoder(w).Encode(result)
}
