package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// UnmuteConversation unmutes notifications for a conversation
func (h *ConversationHandler) UnmuteConversation(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
