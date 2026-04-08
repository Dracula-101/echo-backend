package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// AddParticipants adds participants to a conversation
func (h *ConversationHandler) AddParticipants(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
