package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// RemoveParticipant removes a participant from a conversation
func (h *ConversationHandler) RemoveParticipant(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
