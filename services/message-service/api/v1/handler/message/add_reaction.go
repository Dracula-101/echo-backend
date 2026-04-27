package message

import (
	"net/http"

	"echo-backend/services/message-service/api/v1/dto"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) AddReaction(handler *request.RequestHandler) {
	messageIDStr := handler.PathParam("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		h.log.Warn("Invalid message ID in AddReaction", logger.String("message_id", messageIDStr))
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

	req := dto.NewAddReactionRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	emojiPtr := &req.ReactionType
	appErr := h.reactionService.AddReaction(handler.Context(), messageID, userID, req.ReactionType, emojiPtr, req.SkinTone)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(),
		http.StatusCreated, "Reaction added", nil)
}
