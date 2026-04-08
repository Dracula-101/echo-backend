package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// GetPollResults retrieves the results of a poll
func (h *MessageHandler) GetPollResults(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
