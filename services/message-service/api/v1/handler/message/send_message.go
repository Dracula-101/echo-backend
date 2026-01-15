package message

import (
	"net/http"

	"echo-backend/services/message-service/api/v1/dto"
	svcmodel "echo-backend/services/message-service/internal/service/model"
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
		logger.String("conversation_id", request.ConversationID),
		logger.String("message_type", request.MessageType),
	)

	// Parse parent message ID if provided
	var parentMessageID *uuid.UUID
	if request.ParentMessageID != nil {
		parsed := uuid.MustParse(*request.ParentMessageID)
		parentMessageID = &parsed
	}

	var metadata svcmodel.Metadata
	if request.Metadata != nil {
		metadata = svcmodel.Metadata(request.Metadata)
	}

	// Call service layer
	message, err := h.messageService.SendMessage(handler.Context(), &svcmodel.SendMessageInput{
		ConversationID:  uuid.MustParse(request.ConversationID),
		SenderUserID:    uuid.MustParse(userID),
		Content:         request.Content,
		MessageType:     request.MessageType,
		ParentMessageID: parentMessageID,
		Mentions:        request.Mentions,
		Metadata:        metadata,
	})

	if err != nil {
		h.log.Error("Failed to send message",
			logger.String("user_id", userID),
			logger.String("conversation_id", request.ConversationID),
			logger.Error(err),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Failed to send message", err)
		return
	}

	h.log.Info("Message sent successfully",
		logger.String("user_id", userID),
		logger.String("message_id", message.ID.String()),
		logger.String("conversation_id", message.ConversationID.String()),
	)

	// Send response
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusCreated, "Message sent successfully",
		dto.NewSendMessageResponse(message),
	)
}
