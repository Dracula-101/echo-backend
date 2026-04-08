package message

import (
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetReactions retrieves all reactions for a message
func (h *MessageHandler) GetReactions(handler *request.RequestHandler) {
	messageIDStr := chi.URLParam(handler.Request(), "id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		h.log.Warn("Invalid message ID in GetReactions", logger.String("message_id", messageIDStr))
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid message ID format", nil)
		return
	}

	reactions, appErr := h.reactionService.GetMessageReactions(handler.Context(), messageID)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	// We can group reactions by emoji type for the client or just return array. Let's return the raw array.
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Reactions fetched successfully", map[string]interface{}{
		"reactions": reactions,
	})
}
