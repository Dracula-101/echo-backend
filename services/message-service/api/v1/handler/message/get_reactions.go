package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// GetReactions retrieves all reactions for a message
func (h *MessageHandler) GetReactions(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
