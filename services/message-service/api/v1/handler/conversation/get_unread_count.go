package conversation

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) GetUnreadCount(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	convUUID, err := uuid.Parse(handler.PathParam("id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", err)
		return
	}

	h.log.Debug("Getting unread count",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
	)

	count, svcErr := h.conversationService.GetUnreadCount(ctx, convUUID, userID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Unread count retrieved",
		map[string]interface{}{
			"conversation_id": convUUID.String(),
			"unread_count":    count,
			"mention_count":   0,
		},
	)
}
