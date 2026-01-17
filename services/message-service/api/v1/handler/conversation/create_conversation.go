package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) CreateConversation(handler *request.RequestHandler) {
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Create conversation request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	userID, ok := request.GetUserIDFromContext(handler.Context())
	if !ok {
		h.log.Warn("User not authenticated",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	// Parse and validate request
	req := dto.NewCreateConversationRequest()
	if !handler.ParseValidateAndSend(req) {
		h.log.Warn("Create conversation validation failed",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
		return
	}

	h.log.Debug("Creating conversation",
		logger.String("user_id", userID),
		logger.Int("participant_count", len(req.ParticipantIDs)),
	)

	// Call service layer
	conversation, svcErr := h.conversationService.CreateConversation(
		handler.Context(),
		domain.CreateConversationInput{
			CreatorUserID:    uuid.MustParse(userID),
			ConversationType: domain.ConversationType(req.ConversationType),
			ParticipantIDs: func() []uuid.UUID {
				ids := make([]uuid.UUID, 0, len(req.ParticipantIDs))
				for _, idStr := range req.ParticipantIDs {
					ids = append(ids, uuid.MustParse(idStr))
				}
				return ids
			}(),
			IsPublic:    req.IsPublic,
			Title:       req.Title,
			Description: req.Description,
			AvatarURL:   req.AvatarURL,
			IsEncrypted: req.IsEncrypted,
		},
	)
	if svcErr != nil {
		h.log.Error("Failed to create conversation",
			logger.String("user_id", userID),
			logger.Error(svcErr.Error),
		)
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to create conversation", svcErr.Error)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(),
		response.StatusOK,
		"Conversation created successfully",
		dto.CreateConversationResponse{
			ID:               conversation.ID.String(),
			ConversationType: string(conversation.ConversationType),
			Title:            conversation.Title,
			Description:      conversation.Description,
			CreatorUserID:    conversation.CreatorUserID.String(),
			ParticipantIDs: func() []string {
				ids := make([]string, 0, len(conversation.Participants))
				for _, id := range conversation.Participants {
					ids = append(ids, id.UserID.String())
				}
				return ids
			}(),
			MemberCount: conversation.MemberCount,
			IsPublic:    conversation.IsPublic,
			AvatarURL:   conversation.AvatarURL,
			IsEncrypted: conversation.IsEncrypted,
		},
	)
	h.log.Info("Conversation created successfully",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("conversation_id", conversation.ID.String()),
		logger.String("user_id", userID),
	)
}
