package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// EditMessage updates the content of an existing message
func (h *MessageHandler) EditMessage(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()

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

	request := dto.NewEditMessageRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	h.log.Debug("Editing message",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	msg, err := h.messageService.EditMessage(ctx, messageID, userID, request.Content)
	if err != nil {
		h.log.Error("Failed to edit message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.String("code", err.Code.String()),
		)
		switch err.Code {
		case msgError.CodeMessageNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
		case msgError.CodeMessageNotEditable, msgError.CodeMessageEditExpired:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Message edited successfully", dto.NewSendMessageResponse(msg))
}
