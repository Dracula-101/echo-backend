package conversation

import (
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// RemoveParticipant removes a participant from a conversation
func (h *ConversationHandler) RemoveParticipant(handler *request.RequestHandler) {
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

	participantID := handler.PathParam("user_id")
	if participantID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Participant user ID is required", nil)
		return
	}
	partUUID, err := uuid.Parse(participantID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Participant user ID must be a valid UUID", nil)
		return
	}

	h.log.Debug("Removing participant",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
		logger.String("participant_id", participantID),
	)

	svcErr := h.conversationService.RemoveParticipant(ctx, domain.RemoveParticipantInput{
		ConversationID: convUUID,
		UserID:         uuid.MustParse(userID),
		ParticipantID:  partUUID,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeCannotRemoveOwner:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeDirectConversationLimit:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Participant removed successfully", nil)
}
