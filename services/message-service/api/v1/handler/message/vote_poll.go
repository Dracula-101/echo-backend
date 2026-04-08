package message

import (
	"net/http"
	"shared/server/request"
	"shared/server/response"
)

// VotePoll casts a vote on a poll
func (h *MessageHandler) VotePoll(handler *request.RequestHandler) {
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusNotImplemented, "Not implemented", nil)
}
