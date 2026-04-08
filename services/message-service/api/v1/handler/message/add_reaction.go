package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AddReaction adds a reaction to a message
func (h *MessageHandler) AddReaction(handler *request.RequestHandler) {
	messageIDStr := chi.URLParam(handler.Request(), "id")
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
	userID := uuid.MustParse(userIDStr)

	req := dto.NewAddReactionRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	appErr := h.reactionService.AddReaction(handler.Context(), messageID, userID, req.Emoji, &req.Emoji, nil)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Reaction added successfully", nil)
}
