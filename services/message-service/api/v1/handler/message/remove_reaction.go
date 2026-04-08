package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// RemoveReaction removes a reaction from a message
func (h *MessageHandler) RemoveReaction(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
