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

func (h *ConversationHandler) UpdateParticipantRole(handler *request.RequestHandler) {
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
	targetUUID, err := uuid.Parse(handler.PathParam("user_id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Target user ID must be a valid UUID", err)
		return
	}

	req := dto.NewUpdateParticipantRoleRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	h.log.Debug("Updating participant role",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
		logger.String("target_user_id", targetUUID.String()),
		logger.String("new_role", req.Role),
	)

	svcErr := h.conversationService.UpdateParticipantRole(ctx, domain.UpdateParticipantRoleInput{
		ConversationID: convUUID,
		CallerUserID:   userID,
		TargetUserID:   targetUUID,
		NewRole:        req.Role,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Participant role updated", nil)
}
