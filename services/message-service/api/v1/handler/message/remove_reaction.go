package message

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) RemoveReaction(handler *request.RequestHandler) {
	messageIDStr := handler.PathParam("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		h.log.Warn("Invalid message ID in RemoveReaction", logger.String("message_id", messageIDStr))
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid message ID format", nil)
		return
	}

	reactionType := handler.PathParam("reaction_id")

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

	if appErr := h.reactionService.RemoveReaction(handler.Context(), messageID, userID, reactionType); appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
