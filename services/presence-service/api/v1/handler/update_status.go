package handler

import (
	"net/http"

	"shared/server/response"
)

func (h *PresenceHandler) UpdateCustomStatus(w http.ResponseWriter, r *http.Request) {
	response.JSONWithMessage(r.Context(), r, w, http.StatusNotImplemented, "Not implemented", nil)
}
