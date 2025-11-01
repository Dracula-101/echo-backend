package health

import (
	"net/http"
	"shared/server/env"
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
	isDev := env.IsDevelopment()
	liveness := h.manager.Liveness(r.Context())
	readiness := h.manager.Readiness(r.Context())

	var detailed Response
	if isDev {
		detailed = h.manager.Detailed(r.Context())
	} else {
		detailed = h.manager.Health(r.Context(), false)
	}

	resp := map[string]interface{}{
		"health":    detailed,
		"liveness":  liveness,
		"readiness": readiness,
	}

	status := h.manager.HTTPStatus(detailed.Status)
	if h.manager.HTTPStatus(readiness.Status) > status {
		status = h.manager.HTTPStatus(readiness.Status)
	}
	if h.manager.HTTPStatus(detailed.Status) > status {
		status = h.manager.HTTPStatus(detailed.Status)
	}

	response.JSONWithMessage(r.Context(), r, w, status, "Combined health status", resp)
}
