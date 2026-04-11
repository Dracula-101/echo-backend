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

// UpdateConversation updates a conversation's details
func (h *ConversationHandler) UpdateConversation(handler *request.RequestHandler) {
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

	req := dto.NewUpdateConversationRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	h.log.Debug("Updating conversation",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
	)

	conversation, svcErr := h.conversationService.UpdateConversation(ctx, convUUID, uuid.MustParse(userID), domain.UpdateConversationInput{
		Title:       req.Title,
		Description: req.Description,
		AvatarURL:   req.AvatarURL,
		IsPublic:    req.IsPublic,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodeConversationValidationFailed:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeDirectConversationLimit:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeConversationNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), svcErr.Message)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Conversation updated successfully",
		map[string]interface{}{
			"id":                conversation.ID.String(),
			"title":             conversation.Title,
			"description":       conversation.Description,
			"avatar_url":        conversation.AvatarURL,
			"is_public":         conversation.IsPublic,
			"conversation_type": string(conversation.ConversationType),
		},
	)
}
