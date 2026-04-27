package conversation

import (
	"time"

	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) UpdateConversation(handler *request.RequestHandler) {
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

	req := dto.NewUpdateConversationRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	h.log.Debug("Updating conversation",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
	)

	var inviteExpires *time.Time
	if req.InviteLinkExpiresAt != nil {
		t, perr := time.Parse(time.RFC3339, *req.InviteLinkExpiresAt)
		if perr != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "invite_link_expires_at must be a valid RFC3339 timestamp", perr)
			return
		}
		inviteExpires = &t
	}

	conversation, svcErr := h.conversationService.UpdateConversation(ctx, convUUID, userID, domain.UpdateConversationInput{
		Title:                req.Title,
		Description:          req.Description,
		AvatarURL:            req.AvatarURL,
		IsPublic:             req.IsPublic,
		WhoCanSendMessages:   req.WhoCanSendMessages,
		WhoCanAddMembers:     req.WhoCanAddMembers,
		WhoCanEditInfo:       req.WhoCanEditInfo,
		WhoCanPinMessages:    req.WhoCanPinMessages,
		InviteLinkExpiresAt:  inviteExpires,
		JoinApprovalRequired: req.JoinApprovalRequired,
	})
	if svcErr != nil {
		switch svcErr.Code {
		case msgError.CodeConversationValidationFailed, msgError.CodeDirectConversationLimit:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodePermissionDenied:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		case msgError.CodeConversationNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), svcErr.Message)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Conversation updated",
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
