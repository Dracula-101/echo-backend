package conversation

import (
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) DeleteConversation(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	conversationID := handler.PathParam("id")
	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", err)
		return
	}

	h.log.Debug("Deleting conversation",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID),
	)

	svcErr := h.conversationService.DeleteConversation(ctx, convUUID, userID)
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeConversationNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), svcErr.Message)
		case msgError.CodeDirectConversationLimit:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
