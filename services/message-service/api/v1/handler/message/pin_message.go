package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// PinMessage pins a message in its conversation
func (h *MessageHandler) PinMessage(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
