package conversation

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) GetUserConversations(handler *request.RequestHandler) {
	context := handler.Context()
	userIdInContext, _ := request.GetUserIDFromContext(context)
	userID, uuidErr := uuid.Parse(userIdInContext)
	if uuidErr != nil {
		h.log.Error("Invalid user ID in context",
			logger.String("user_id", userID.String()),
			logger.Error(uuidErr),
		)
		response.BadRequestError(context, handler.Request(), handler.Writer(), "Invalid user ID", uuidErr)
		return
	}

	conversations, err := h.conversationService.GetUserConversations(context, userID)
	if err != nil {
		h.log.Error("Failed to get user conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err.Error),
		)
		response.InternalServerError(context, handler.Request(), handler.Writer(), "Failed to get user conversations", err.Error)
		return
	}

	response.JSONWithMessage(context, handler.Request(), handler.Writer(), response.StatusOK, "User conversations fetched successfully", conversations)
}
