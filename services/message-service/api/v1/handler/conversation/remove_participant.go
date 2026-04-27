package conversation

import (
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) RemoveParticipant(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	convUUID, err := uuid.Parse(handler.PathParam("id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", err)
		return
	}
	partUUID, err := uuid.Parse(handler.PathParam("user_id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Participant user ID must be a valid UUID", err)
		return
	}

	h.log.Debug("Removing participant",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
		logger.String("participant_id", partUUID.String()),
	)

	svcErr := h.conversationService.RemoveParticipant(ctx, domain.RemoveParticipantInput{
		ConversationID: convUUID,
		UserID:         userID,
		ParticipantID:  partUUID,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeCannotRemoveOwner, msgError.CodeDirectConversationLimit:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
