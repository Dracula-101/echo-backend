package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// UnpinMessage unpins a message from its conversation
func (h *MessageHandler) UnpinMessage(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
