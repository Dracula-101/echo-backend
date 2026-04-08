package message

import (
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RemoveReaction removes a reaction from a message
func (h *MessageHandler) RemoveReaction(handler *request.RequestHandler) {
	messageIDStr := chi.URLParam(handler.Request(), "id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		h.log.Warn("Invalid message ID in RemoveReaction", logger.String("message_id", messageIDStr))
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid message ID format", nil)
		return
	}

	// Reaction type is passed as `{reaction_id}` parameter according to main.go routing.
	// r.Delete("/{id}/reactions/{reaction_id}", ...)
	reactionType := chi.URLParam(handler.Request(), "reaction_id")

	userIDStr, ok := request.GetUserIDFromContext(handler.Context())
	if !ok {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}
	userID := uuid.MustParse(userIDStr)

	appErr := h.reactionService.RemoveReaction(handler.Context(), messageID, userID, reactionType)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusOK, "Reaction removed successfully", nil)
}
