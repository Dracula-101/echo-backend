package message

import (
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// PinMessage pins a message in its conversation
func (h *MessageHandler) PinMessage(handler *req.RequestHandler) {
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

	h.log.Debug("Pinning message",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	err := h.messageService.PinMessage(ctx, messageID, userID)
	if err != nil {
		h.log.Error("Failed to pin message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.String("code", err.Code.String()),
		)
		switch err.Code {
		case msgError.CodeMessageNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
		case msgError.CodeAlreadyPinned:
			response.ConflictError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusCreated, "Message pinned", nil)
}
