package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// GetDeliveryStatus retrieves the delivery status of a message
func (h *MessageHandler) GetDeliveryStatus(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
