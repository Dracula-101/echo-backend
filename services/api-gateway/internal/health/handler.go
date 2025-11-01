package health

import (
	"net/http"
	"shared/server/response"
)

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	includeChecks := r.URL.Query().Get("checks") == "true"
	resp := h.manager.Health(r.Context(), includeChecks)
	response.JSONWithMessage(r.Context(), r, w, h.manager.HTTPStatus(resp.Status), "Health status", resp)
}

func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	resp := h.manager.Liveness(r.Context())
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Liveness status", resp)
}

func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	resp := h.manager.Readiness(r.Context())
	response.JSONWithMessage(r.Context(), r, w, h.manager.HTTPStatus(resp.Status), "Readiness status", resp)
}

func (h *Handler) Detailed(w http.ResponseWriter, r *http.Request) {
	resp := h.manager.Detailed(r.Context())
	response.JSONWithMessage(r.Context(), r, w, h.manager.HTTPStatus(resp.Status), "Detailed status", resp)
}
