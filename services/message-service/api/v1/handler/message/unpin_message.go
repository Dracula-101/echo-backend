package message

import (
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// UnpinMessage unpins a message from its conversation
func (h *MessageHandler) UnpinMessage(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	messageID := handler.PathParam("id")
	if messageID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID is required", nil)
		return
	}

	h.log.Debug("Unpinning message",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	err := h.messageService.UnpinMessage(ctx, messageID, userID)
	if err != nil {
		switch err.Code {
		case msgError.CodeMessageNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
		case msgError.CodeNotPinned:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		}
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
