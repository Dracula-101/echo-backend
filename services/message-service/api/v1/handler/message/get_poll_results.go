package message

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) GetPollResults(handler *request.RequestHandler) {
	pollIDStr := handler.PathParam("id")
	pollID, err := uuid.Parse(pollIDStr)
	if err != nil {
		h.log.Warn("Invalid poll ID in GetPollResults", logger.String("poll_id", pollIDStr))
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid poll ID format", nil)
		return
	}

	options, totalVotes, appErr := h.pollService.GetPollResults(handler.Context(), pollID)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Poll results", map[string]interface{}{
		"poll_id":     pollIDStr,
		"options":     options,
		"total_votes": totalVotes,
	})
}
