package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// DeleteMessage deletes a message by its ID
func (h *MessageHandler) DeleteMessage(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
