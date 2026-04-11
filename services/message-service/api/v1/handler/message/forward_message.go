package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// ForwardMessage forwards a message to another conversation
func (h *MessageHandler) ForwardMessage(handler *req.RequestHandler) {
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

	request := dto.NewForwardMessageRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	h.log.Debug("Forwarding message",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
		logger.String("target_conversation_id", request.TargetConversationID),
	)

	msg, err := h.messageService.ForwardMessage(ctx, messageID, userID, request.TargetConversationID, handler.GetDeviceInfo(), handler.GetClientIP())
	if err != nil {
		h.log.Error("Failed to forward message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.String("code", err.Code.String()),
		)
		switch err.Code {
		case msgError.CodeMessageNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
		case msgError.CodeTargetConversationNotFound:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Message forwarded successfully", dto.NewSendMessageResponse(msg))
}
