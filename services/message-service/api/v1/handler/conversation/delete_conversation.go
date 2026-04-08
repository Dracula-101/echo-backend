package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// DeleteConversation deletes or leaves a conversation
func (h *ConversationHandler) DeleteConversation(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
