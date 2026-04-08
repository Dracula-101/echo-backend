package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// ForwardMessage forwards a message to another conversation
func (h *MessageHandler) ForwardMessage(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
