package message

import (
	"net/http"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) GetReactions(handler *request.RequestHandler) {
	messageIDStr := handler.PathParam("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		h.log.Warn("Invalid message ID in GetReactions", logger.String("message_id", messageIDStr))
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid message ID format", nil)
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

	reactions, summary, appErr := h.reactionService.GetMessageReactionsWithSummary(handler.Context(), messageID, userID)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Reactions fetched", map[string]interface{}{
		"summary":   summary,
		"reactions": reactions,
	})
}
