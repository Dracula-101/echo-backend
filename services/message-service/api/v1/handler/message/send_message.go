package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// SendMessage flow at a glance:
//
//	[1] Request helper — parse + validate JSON, capture request/correlation IDs.
//	[2] Auth middleware context — ensure caller has a user ID.
//	[3] Metadata shaping — map incoming metadata into internal model (MediaURL, etc.).
//	[4] MessageService.SendMessage — persist content, apply threading/mentions logic.
//	[5] Response helper — return 201 with DTO payload.
//
// Every step logs request + user identifiers for traceability.
func (h *MessageHandler) SendMessage(handler *request.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Send message request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	// Extract user_id from context (set by auth middleware in API Gateway)
	userID, ok := req.GetUserIDFromContext(handler.Context())
	if !ok {
		h.log.Warn("User not authenticated",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	idempotencyKey, _ := req.GetIdempotencyKeyFromContext(handler.Context())
	h.log.Debug("Idempotency key extracted",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
		logger.String("idempotency_key", idempotencyKey),
	)
	h.log.Debug("Acquiring message lock",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
	)
	h.messageCache.AcquireMessageLock(ctx, idempotencyKey)
	defer h.messageCache.ReleaseMessageLock(ctx, idempotencyKey)

	conversationId, err := uuid.Parse(handler.PathParam("conversation_id"))
	if err != nil {
		h.log.Warn("Invalid conversation ID format",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.String("conversation_id", handler.PathParam("conversation_id")),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Invalid conversation ID format", nil)
		return
	}

	// Parse and validate request
	request := dto.NewSendMessageRequest()
	if !handler.ParseValidateAndSend(request) {
		h.log.Warn("Send message validation failed",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
		return
	}

	h.log.Debug("Sending message",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationId.String()),
		logger.String("message_type", string(request.MessageType)),
	)

	// Call service layer
	msg, msgErr := h.messageService.SendMessage(
		handler.Context(),
		&domain.SendMessageInput{
			ConversationID:  conversationId,
			SenderUserID:    uuid.MustParse(userID),
			Content:         request.Content,
			MessageType:     domain.MessageType(request.MessageType),
			ParentMessageID: request.ParentMessageID,
		},
		handler.GetDeviceInfo(),
		handler.GetClientIP(),
	)

	if msgErr != nil {
		h.log.Error("Failed to send message",
			logger.String("user_id", userID),
			logger.String("conversation_id", conversationId.String()),
			logger.Error(msgErr.Error),
		)
		switch msgErr.Code {
		case msgError.CodeUserBlocked, msgError.CodeParticipantNotAllowedToSendMessages, msgError.CodeConversationInactive:
			response.ForbiddenError(handler.Context(), handler.Request(), handler.Writer(), msgErr.Message, msgErr.Error)
		case msgError.CodeRateLimitExceeded:
			response.TooManyRequestsError(handler.Context(), handler.Request(), handler.Writer(), msgErr.Message, 100)
		case msgError.CodeParticipantNotInConversation, msgError.CodeBlockedUserNotFound, msgError.CodeParticipantRemovedFromConversation,
			msgError.CodeParticipantLeftConversation, msgError.CodeEmptyMessageContent, msgError.CodeMissingMessageMetadata, msgError.CodeConversationValidationFailed:
			response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), msgErr.Message, msgErr.Error)
		default:
			response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to send message", msgErr.Error)
		}
		return
	}

	h.log.Info("Message sent successfully",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationId.String()),
	)

	// Send response
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(),
		response.StatusOK, "Message sent successfully", dto.NewSendMessageResponse(msg),
	)
}
