package conversation

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetUnreadCount retrieves the unread message count for a conversation
func (h *ConversationHandler) GetUnreadCount(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	conversationID := handler.PathParam("id")
	if conversationID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID is required", nil)
		return
	}
	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", nil)
		return
	}

	h.log.Debug("Getting unread count",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
	)

	count, svcErr := h.conversationService.GetUnreadCount(ctx, convUUID, uuid.MustParse(userID))
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Unread count retrieved",
		map[string]interface{}{
			"conversation_id": conversationID,
			"unread_count":    count,
		},
	)
}
