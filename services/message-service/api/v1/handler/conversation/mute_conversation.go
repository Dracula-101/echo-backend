package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// MuteConversation mutes notifications for a conversation
func (h *ConversationHandler) MuteConversation(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
