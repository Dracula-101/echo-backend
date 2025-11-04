package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"echo-backend/services/message-service/internal/model"
	"echo-backend/services/message-service/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"shared/pkg/logger"
)

// HTTPHandler handles HTTP requests for the message service
type HTTPHandler struct {
	messageService service.MessageService
	logger         logger.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(messageService service.MessageService, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		messageService: messageService,
		logger:         logger,
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendMessage handles POST /messages
func (h *HTTPHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Parse request body
	var req model.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Set sender user ID from authenticated user
	req.SenderUserID = userID

	// Validate request
	if err := h.validateSendMessageRequest(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Send message
	message, err := h.messageService.SendMessage(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to send message",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		h.respondError(w, http.StatusInternalServerError, "send_failed", err.Error())
		return
	}

	h.respondSuccess(w, http.StatusCreated, message)
}

// GetMessages handles GET /conversations/{conversation_id}/messages
func (h *HTTPHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Get conversation ID from URL
	vars := mux.Vars(r)
	conversationID, err := uuid.Parse(vars["conversation_id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_conversation_id", "Invalid conversation ID")
		return
	}

	// Parse pagination parameters
	params := &model.PaginationParams{
		Limit: 50, // default
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}

	if beforeID := r.URL.Query().Get("before_id"); beforeID != "" {
		if id, err := uuid.Parse(beforeID); err == nil {
			params.BeforeID = &id
		}
	}

	if afterID := r.URL.Query().Get("after_id"); afterID != "" {
		if id, err := uuid.Parse(afterID); err == nil {
			params.AfterID = &id
		}
	}

	// Get messages
	response, err := h.messageService.GetMessages(ctx, conversationID, params)
	if err != nil {
		h.logger.Error("Failed to get messages",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		h.respondError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	h.respondSuccess(w, http.StatusOK, response)
}

// GetMessage handles GET /messages/{message_id}
func (h *HTTPHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	_, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Get message ID from URL
	vars := mux.Vars(r)
	messageID, err := uuid.Parse(vars["message_id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_message_id", "Invalid message ID")
		return
	}

	// Get message
	message, err := h.messageService.GetMessage(ctx, messageID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "not_found", "Message not found")
		return
	}

	h.respondSuccess(w, http.StatusOK, message)
}

// EditMessage handles PUT /messages/{message_id}
func (h *HTTPHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Get message ID from URL
	vars := mux.Vars(r)
	messageID, err := uuid.Parse(vars["message_id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_message_id", "Invalid message ID")
		return
	}

	// Parse request body
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.Content == "" {
		h.respondError(w, http.StatusBadRequest, "empty_content", "Content cannot be empty")
		return
	}

	// Edit message
	if err := h.messageService.EditMessage(ctx, messageID, userID, req.Content); err != nil {
		h.logger.Error("Failed to edit message",
			logger.String("user_id", userID.String()),
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		h.respondError(w, http.StatusInternalServerError, "edit_failed", err.Error())
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"message": "Message updated successfully",
	})
}

// DeleteMessage handles DELETE /messages/{message_id}
func (h *HTTPHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Get message ID from URL
	vars := mux.Vars(r)
	messageID, err := uuid.Parse(vars["message_id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_message_id", "Invalid message ID")
		return
	}

	// Delete message
	if err := h.messageService.DeleteMessage(ctx, messageID, userID); err != nil {
		h.logger.Error("Failed to delete message",
			logger.String("user_id", userID.String()),
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		h.respondError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"message": "Message deleted successfully",
	})
}

// MarkAsRead handles POST /messages/{message_id}/read
func (h *HTTPHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Get message ID from URL
	vars := mux.Vars(r)
	messageID, err := uuid.Parse(vars["message_id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_message_id", "Invalid message ID")
		return
	}

	// Mark as read
	if err := h.messageService.HandleReadReceipt(ctx, userID, messageID); err != nil {
		h.logger.Error("Failed to mark message as read",
			logger.String("user_id", userID.String()),
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		h.respondError(w, http.StatusInternalServerError, "mark_read_failed", err.Error())
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"message": "Message marked as read",
	})
}

// SetTyping handles POST /conversations/{conversation_id}/typing
func (h *HTTPHandler) SetTyping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Get conversation ID from URL
	vars := mux.Vars(r)
	conversationID, err := uuid.Parse(vars["conversation_id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_conversation_id", "Invalid conversation ID")
		return
	}

	// Parse request body
	var req struct {
		IsTyping bool `json:"is_typing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Set typing indicator
	if err := h.messageService.SetTypingIndicator(ctx, conversationID, userID, req.IsTyping); err != nil {
		h.logger.Error("Failed to set typing indicator",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		h.respondError(w, http.StatusInternalServerError, "typing_failed", err.Error())
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"message": "Typing indicator updated",
	})
}

// Health handles GET /health
func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "message-service",
	})
}

// Helper methods

func (h *HTTPHandler) respondSuccess(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

func (h *HTTPHandler) respondError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

func (h *HTTPHandler) validateSendMessageRequest(req *model.SendMessageRequest) error {
	if req.ConversationID == uuid.Nil {
		return &ValidationError{Field: "conversation_id", Message: "conversation_id is required"}
	}

	if req.Content == "" {
		return &ValidationError{Field: "content", Message: "content is required"}
	}

	if len(req.Content) > 10000 {
		return &ValidationError{Field: "content", Message: "content too long (max 10000 characters)"}
	}

	validTypes := map[string]bool{
		"text":     true,
		"image":    true,
		"video":    true,
		"audio":    true,
		"file":     true,
		"location": true,
		"poll":     true,
	}

	if !validTypes[req.MessageType] {
		return &ValidationError{Field: "message_type", Message: "invalid message_type"}
	}

	return nil
}

func getUserIDFromContext(r *http.Request) (uuid.UUID, error) {
	// Try to get from context (set by auth middleware)
	if userIDVal := r.Context().Value("user_id"); userIDVal != nil {
		if userIDStr, ok := userIDVal.(string); ok {
			return uuid.Parse(userIDStr)
		}
		if userID, ok := userIDVal.(uuid.UUID); ok {
			return userID, nil
		}
	}

	// Fallback: get from header (for testing)
	if userIDStr := r.Header.Get("X-User-ID"); userIDStr != "" {
		return uuid.Parse(userIDStr)
	}

	return uuid.Nil, &AuthError{Message: "user_id not found in context"}
}

// Custom errors
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
