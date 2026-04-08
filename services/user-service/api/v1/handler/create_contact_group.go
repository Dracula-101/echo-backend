package handler

import (
	"net/http"
	req "shared/server/request"
	"shared/server/response"
)

func (h *UserHandler) CreateContactGroup(handler *req.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
