package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *MessageHandler) DeleteMessage(handler *req.RequestHandler) {
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

	body := dto.NewDeleteMessageRequest()
	handler.AllowEmptyBody()
	if !handler.ParseValidateAndSend(body) {
		return
	}
	deleteForEveryone := body.DeleteFor == "" || body.DeleteFor == "everyone"

	h.log.Debug("Deleting message",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
		logger.Bool("delete_for_everyone", deleteForEveryone),
	)

	err := h.messageService.DeleteMessage(ctx, messageID, userID, deleteForEveryone)
	if err != nil {
		h.log.Error("Failed to delete message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.String("code", err.Code.String()),
		)
		switch err.Code {
		case msgError.CodeMessageNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
		case msgError.CodeMessageNotDeletable:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		}
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
