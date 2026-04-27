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

func (h *ConversationHandler) AddParticipants(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	conversationID := handler.PathParam("id")
	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", err)
		return
	}

	req := dto.NewAddParticipantsRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	participantIDs := make([]uuid.UUID, 0, len(req.ParticipantIDs))
	for _, idStr := range req.ParticipantIDs {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "participant_ids must be valid UUIDs", parseErr)
			return
		}
		participantIDs = append(participantIDs, id)
	}

	h.log.Debug("Adding participants",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID),
		logger.Int("count", len(participantIDs)),
	)

	svcErr := h.conversationService.AddParticipants(ctx, domain.AddParticipantsInput{
		ConversationID: convUUID,
		UserID:         userID,
		ParticipantIDs: participantIDs,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeDirectConversationLimit, msgError.CodeMaxMembersExceeded:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusCreated, "Participants added", nil)
}
