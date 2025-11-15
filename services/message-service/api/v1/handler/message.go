package handler

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/models"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

// GetMessages handles retrieving messages for a conversation
func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get messages request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	request := dto.NewGetMessagesRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	messages, err := h.service.GetMessages(r.Context(), uuid.MustParse(request.ConversationID), &models.PaginationParams{
		Limit: request.Limit,
	})
	if err != nil {
		h.log.Error("Failed to get messages",
			logger.String("conversation_id", request.ConversationID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(r.Context(), r, w, "Failed to get messages", err)
		return
	}

	responseDTO := dto.GetMessagesResponse{
		MessagesResponse: *messages,
	}

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Messages retrieved successfully", responseDTO)
}

// EditMessage handles editing an existing message
func (h *MessageHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Edit message request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Get message ID from path
	vars := mux.Vars(r)
	messageID := vars["id"]
	if messageID == "" {
		response.BadRequestError(r.Context(), r, w, "Message ID is required", nil)
		return
	}

	// Parse and validate request
	request := dto.NewEditMessageRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	err := h.service.EditMessage(r.Context(), uuid.MustParse(messageID), uuid.MustParse(userID), request.Content)
	if err != nil {
		h.log.Error("Failed to edit message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to edit message", err)
		return
	}

	h.log.Info("Message edited successfully",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Message edited successfully", nil)
}

// DeleteMessage handles deleting a message
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Delete message request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Get message ID from path
	vars := mux.Vars(r)
	messageID := vars["id"]
	if messageID == "" {
		response.BadRequestError(r.Context(), r, w, "Message ID is required", nil)
		return
	}

	// Call service layer
	err := h.service.DeleteMessage(r.Context(), uuid.MustParse(messageID), uuid.MustParse(userID))
	if err != nil {
		h.log.Error("Failed to delete message",
			logger.String("user_id", userID),
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to delete message", err)
		return
	}

	h.log.Info("Message deleted successfully",
		logger.String("user_id", userID),
		logger.String("message_id", messageID),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Message deleted successfully", nil)
}

// MarkAsRead handles marking a message as read
func (h *MessageHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Mark as read request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Parse and validate request
	request := dto.NewMarkAsReadRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	err := h.service.MarkAsRead(r.Context(), uuid.MustParse(request.MessageID), uuid.MustParse(userID))
	if err != nil {
		h.log.Error("Failed to mark message as read",
			logger.String("user_id", userID),
			logger.String("message_id", request.MessageID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to mark as read", err)
		return
	}

	h.log.Info("Message marked as read",
		logger.String("user_id", userID),
		logger.String("message_id", request.MessageID),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Message marked as read", nil)
}

// SetTypingIndicator handles setting typing indicator
func (h *MessageHandler) SetTypingIndicator(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Parse and validate request
	request := dto.NewTypingIndicatorRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	err := h.service.SetTypingIndicator(r.Context(), uuid.MustParse(request.ConversationID), uuid.MustParse(userID), request.IsTyping)
	if err != nil {
		h.log.Error("Failed to set typing indicator",
			logger.String("user_id", userID),
			logger.String("conversation_id", request.ConversationID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to set typing indicator", err)
		return
	}

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Typing indicator set", nil)
}
