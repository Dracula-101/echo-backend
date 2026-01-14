package handler

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/models"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// SendMessage handles sending a new message
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Send message request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	// Extract user_id from context (set by auth middleware in API Gateway)
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User not authenticated",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
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

	// Convert metadata map to Metadata struct (simplified - can be enhanced later)
	metadata := models.Metadata{}
	if request.Metadata != nil {
		// Extract known fields from map if they exist
		if mediaURL, ok := request.Metadata["media_url"].(string); ok {
			metadata.MediaURL = mediaURL
		}
		if thumbnailURL, ok := request.Metadata["thumbnail_url"].(string); ok {
			metadata.ThumbnailURL = thumbnailURL
		}
		if fileName, ok := request.Metadata["file_name"].(string); ok {
			metadata.FileName = fileName
		}
		if fileSize, ok := request.Metadata["file_size"].(float64); ok {
			metadata.FileSize = int64(fileSize)
		}
	}

	// Call service layer
	message, err := h.service.SendMessage(r.Context(), &models.SendMessageRequest{
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
		response.InternalServerError(r.Context(), r, w, "Failed to send message", err)
		return
	}

	h.log.Info("Message sent successfully",
		logger.String("user_id", userID),
		logger.String("message_id", message.ID.String()),
		logger.String("conversation_id", message.ConversationID.String()),
	)

	// Send response
	response.JSONWithMessage(r.Context(), r, w, http.StatusCreated, "Message sent successfully",
		dto.NewSendMessageResponse(message),
	)
}
