package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// SearchMessages searches for messages matching a query
func (h *MessageHandler) SearchMessages(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
