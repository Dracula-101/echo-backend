package conversation

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"echo-backend/services/message-service/api/v1/dto"
	convError "echo-backend/services/message-service/internal/error"

	"github.com/google/uuid"
)

func (h *ConversationHandler) GetConversationByID(handler *request.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()
	conversationIDParam := handler.PathParam("id")

	h.log.Info("Get conversation by ID request received",
		logger.String("service", convError.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("conversation_id_param", conversationIDParam),
	)

	// Validate conversation ID format
	conversationID, err := uuid.Parse(conversationIDParam)
	if err != nil {
		h.log.Warn("Invalid conversation_id format",
			logger.String("service", convError.ServiceName),
			logger.String("request_id", requestID),
			logger.String("conversation_id", conversationIDParam),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", nil)
	}

	// Fetch conversation from service
	conversation, appErr := h.conversationService.GetConversation(ctx, conversationID)
	if appErr != nil {
		h.log.Error("Failed to get conversation by ID",
			logger.String("service", convError.ServiceName),
			logger.String("request_id", requestID),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(appErr.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), appErr.Message, appErr.Error)
	}

	h.log.Info("Conversation retrieved successfully",
		logger.String("service", convError.ServiceName),
		logger.String("request_id", requestID),
		logger.String("conversation_id", conversation.ID.String()),
	)

	// Prepare and send response
	resp := dto.ConversationResponse{
		ID:               conversation.ID.String(),
		ConversationType: string(conversation.ConversationType),
		Title:            conversation.Title,
		AvatarURL:        conversation.AvatarURL,
		IsEncrypted:      conversation.IsEncrypted,
		IsPublic:         conversation.IsPublic,
		MemberCount:      conversation.MemberCount,
		UnreadCount:      conversation.UnreadCount,
		LastMessageAt:    conversation.LastMessageAt,
		CreatedAt:        conversation.CreatedAt,
	}
	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Conversation retrieved successfully", resp)
}
