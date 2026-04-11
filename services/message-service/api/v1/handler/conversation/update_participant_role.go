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

// UpdateParticipantRole updates the role of a participant in a conversation
func (h *ConversationHandler) UpdateParticipantRole(handler *request.RequestHandler) {
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

	targetUserID := handler.PathParam("user_id")
	if targetUserID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Target user ID is required", nil)
		return
	}
	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Target user ID must be a valid UUID", nil)
		return
	}

	req := dto.NewUpdateParticipantRoleRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	h.log.Debug("Updating participant role",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
		logger.String("target_user_id", targetUserID),
		logger.String("new_role", req.Role),
	)

	svcErr := h.conversationService.UpdateParticipantRole(ctx, domain.UpdateParticipantRoleInput{
		ConversationID: convUUID,
		CallerUserID:   uuid.MustParse(userID),
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
		response.StatusOK, "Participant role updated successfully", nil)
}
