package conversation

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// GetPinnedMessages retrieves all pinned messages in a conversation
func (h *ConversationHandler) GetPinnedMessages(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
