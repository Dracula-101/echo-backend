package handler

import (
	"echo-backend/services/message-service/api/v1/dto"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	pkgErrors "shared/pkg/errors"

	"github.com/google/uuid"
)

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	service ConversationService
	log     logger.Logger
}

// ConversationService interface for conversation operations
type ConversationService interface {
	CreateConversation(userID uuid.UUID, conversationType string, participantIDs []uuid.UUID, title, description string, isEncrypted, isPublic bool) (uuid.UUID, []uuid.UUID, int64, pkgErrors.AppError)
	GetConversations(userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int, pkgErrors.AppError)
}

func NewConversationHandler(service ConversationService, log logger.Logger) *ConversationHandler {
	return &ConversationHandler{
		service: service,
		log:     log,
	}
}

// CreateConversation handles creating a new conversation
func (h *ConversationHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Create conversation request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	// Extract user_id from context
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		h.log.Warn("User not authenticated",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Parse and validate request
	request := dto.NewCreateConversationRequest()
	if !handler.ParseValidateAndSend(request) {
		h.log.Warn("Create conversation validation failed",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
		return
	}

	h.log.Debug("Creating conversation",
		logger.String("user_id", userID),
		logger.String("conversation_type", request.ConversationType),
		logger.Int("participant_count", len(request.ParticipantIDs)),
	)

	// Parse participant IDs
	participantIDs := make([]uuid.UUID, len(request.ParticipantIDs))
	for i, id := range request.ParticipantIDs {
		participantIDs[i] = uuid.MustParse(id)
	}

	// Call service layer
	conversationID, allParticipants, createdAt, err := h.service.CreateConversation(
		uuid.MustParse(userID),
		request.ConversationType,
		participantIDs,
		request.Title,
		request.Description,
		request.IsEncrypted,
		request.IsPublic,
	)

	if err != nil {
		h.log.Error("Failed to create conversation",
			logger.String("user_id", userID),
			logger.String("conversation_type", request.ConversationType),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to create conversation", err)
		return
	}

	h.log.Info("Conversation created successfully",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID.String()),
		logger.String("conversation_type", request.ConversationType),
	)

	// Send response
	response.JSONWithMessage(r.Context(), r, w, http.StatusCreated, "Conversation created successfully",
		dto.NewCreateConversationResponse(
			conversationID,
			request.ConversationType,
			request.Title,
			request.Description,
			uuid.MustParse(userID),
			request.IsEncrypted,
			request.IsPublic,
			allParticipants,
			createdAt,
		),
	)
}

// GetConversations handles retrieving user's conversations
func (h *ConversationHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.log.Info("Get conversations request received",
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
	request := dto.NewGetConversationsRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	// Call service layer
	conversations, total, err := h.service.GetConversations(
		uuid.MustParse(userID),
		request.Limit,
		request.Offset,
	)

	if err != nil {
		h.log.Error("Failed to get conversations",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to get conversations", err)
		return
	}

	hasMore := request.Offset+len(conversations) < total

	// Send response
	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Conversations retrieved successfully",
		dto.GetConversationsResponse{
			Conversations: conversations,
			Total:         total,
			Limit:         request.Limit,
			Offset:        request.Offset,
			HasMore:       hasMore,
		},
	)
}
