package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// EditMessage updates the content of an existing message
func (h *MessageHandler) EditMessage(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
