package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// UpdateParticipantRole updates the role of a participant in a conversation
func (h *ConversationHandler) UpdateParticipantRole(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
