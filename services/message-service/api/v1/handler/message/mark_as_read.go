package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// MarkAsRead marks a message as read by the current user
func (h *MessageHandler) MarkAsRead(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
