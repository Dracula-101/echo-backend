package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// CreatePoll creates a new poll in a conversation
func (h *MessageHandler) CreatePoll(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
