package message

import (
	"net/http"

	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) VotePoll(handler *request.RequestHandler) {
	pollIDStr := handler.PathParam("id")
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
	userID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Invalid user id", parseErr)
		return
	}

	req := dto.NewVotePollRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	for _, idStr := range req.OptionIDs {
		optID, perr := uuid.Parse(idStr)
		if perr != nil {
			response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "option_ids contains an invalid UUID", perr)
			return
		}
		input := domain.VotePollInput{PollID: pollID, OptionID: optID, UserID: userID}
		if err := h.pollService.VotePoll(handler.Context(), input); err != nil {
			h.sendAppError(handler, err)
			return
		}
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Vote cast", nil)
}
