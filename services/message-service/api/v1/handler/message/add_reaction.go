package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// AddReaction adds a reaction to a message
func (h *MessageHandler) AddReaction(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
