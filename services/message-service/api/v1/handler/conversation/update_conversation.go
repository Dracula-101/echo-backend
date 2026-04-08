package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// UpdateConversation updates a conversation's details
func (h *ConversationHandler) UpdateConversation(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
