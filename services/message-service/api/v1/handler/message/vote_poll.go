package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// VotePoll casts a vote on a poll
func (h *MessageHandler) VotePoll(handler *request.RequestHandler) {
	pollIDStr := chi.URLParam(handler.Request(), "id")
	pollID, err := uuid.Parse(pollIDStr)
	if err != nil {
		h.log.Warn("Invalid poll ID in VotePoll", logger.String("poll_id", pollIDStr))
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid poll ID format", nil)
		return
	}

	userIDStr, ok := request.GetUserIDFromContext(handler.Context())
	if !ok {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	req := dto.NewVotePollRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	// Fetch current options to map OptionIndexes to OptionIDs
	options, _, appErr := h.pollService.GetPollResults(handler.Context(), pollID)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	// For simplicity, handle multiple option votes by looping over them, unless !allow_multiple which PollService handles loosely right now.
	for _, idx := range req.OptionIndexes {
		if idx < 0 || idx >= len(options) {
			response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid option index", nil)
			return
		}
		optID := uuid.MustParse(options[idx].ID)
		
		input := domain.VotePollInput{
			PollID:   pollID,
			OptionID: optID,
			UserID:   uuid.MustParse(userIDStr),
		}

		if err := h.pollService.VotePoll(handler.Context(), input); err != nil {
			h.sendAppError(handler, err)
			return
		}
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Votes cast successfully", nil)
}
