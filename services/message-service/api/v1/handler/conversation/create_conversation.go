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

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	// Parse and validate request
	req := dto.NewCreateConversationRequest()
	if !handler.ParseValidateAndSend(req) {
		h.log.Warn("Create conversation validation failed",
			logger.String("request_id", requestID),
			logger.String("user_id", userID.String()),
		)
		return
	}

	// check if title is empty for group conversations
	if req.ConversationType != dto.ConversationTypeDirect && req.Title == "" {
		h.log.Warn("Title is required for group, channel, and broadcast conversations",
			logger.String("request_id", requestID),
			logger.String("user_id", userID.String()),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Title is required for group, channel, and broadcast conversations", nil)
		return
	}

	if len(req.ParticipantIDs) == 0 {
		h.log.Warn("At least one participant is required",
			logger.String("request_id", requestID),
			logger.String("user_id", userID.String()),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "At least one participant is required", nil)
		return
	}

	if req.ConversationType == dto.ConversationTypeDirect && len(req.ParticipantIDs) != 1 {
		h.log.Warn("Direct conversations must have exactly 2 participants",
			logger.String("request_id", requestID),
			logger.String("user_id", userID.String()),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Direct conversations must have exactly 2 participants", nil)
		return
	}
	if req.MaxMembers != nil && *req.MaxMembers < len(req.ParticipantIDs)+1 {
		h.log.Warn("Max members must be greater than the number of participants",
			logger.String("request_id", requestID),
			logger.String("user_id", userID.String()),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "Max members must be greater than the number of participants", nil)
		return
	}

	h.log.Debug("Creating conversation",
		logger.String("user_id", userID.String()),
		logger.Int("participant_count", len(req.ParticipantIDs)),
	)

	participantUUIDs := make([]uuid.UUID, 0, len(req.ParticipantIDs))
	for _, idStr := range req.ParticipantIDs {
		id, perr := uuid.Parse(idStr)
		if perr != nil {
			response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "participant_ids must be valid UUIDs", perr)
			return
		}
		participantUUIDs = append(participantUUIDs, id)
	}

	conversation, svcErr := h.conversationService.CreateConversation(
		handler.Context(),
		domain.CreateConversationInput{
			CreatorUserID:        userID,
			ConversationType:     domain.ConversationType(req.ConversationType),
			ParticipantIDs:       participantUUIDs,
			IsPublic:             req.IsPublic,
			Title:                req.Title,
			Description:          req.Description,
			AvatarURL:            req.AvatarURL,
			IsEncrypted:          req.IsEncrypted,
			MaxMembers:           req.MaxMembers,
			JoinApprovalRequired: req.JoinApprovalRequired,
			WhoCanSendMessages:   req.WhoCanSendMessages.String(),
			WhoCanAddMembers:     req.WhoCanAddMembers.String(),
			WhoCanEditInfo:       req.WhoCanEditInfo.String(),
			WhoCanPinMessages:    req.WhoCanPinMessages.String(),
		},
	)
	if svcErr != nil {
		h.log.Error("Failed to create conversation",
			logger.String("user_id", userID.String()),
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
		logger.String("user_id", userID.String()),
	)
}
