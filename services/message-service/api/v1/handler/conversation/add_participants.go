package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// AddParticipants adds participants to a conversation
func (h *ConversationHandler) AddParticipants(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	conversationID := handler.PathParam("id")
	if conversationID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID is required", nil)
		return
	}
	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", nil)
		return
	}

	req := dto.NewAddParticipantsRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	participantIDs := make([]uuid.UUID, 0, len(req.ParticipantIDs))
	for _, idStr := range req.ParticipantIDs {
		participantIDs = append(participantIDs, uuid.MustParse(idStr))
	}

	h.log.Debug("Adding participants",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
		logger.Int("count", len(participantIDs)),
	)

	svcErr := h.conversationService.AddParticipants(ctx, domain.AddParticipantsInput{
		ConversationID: convUUID,
		UserID:         uuid.MustParse(userID),
		ParticipantIDs: participantIDs,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeDirectConversationLimit:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeMaxMembersExceeded:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Participants added successfully", nil)
}
