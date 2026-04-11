package conversation

import (
	"time"

	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// CreateInvite creates a conversation invite link
func (h *ConversationHandler) CreateInvite(handler *request.RequestHandler) {
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

	req := dto.NewCreateInviteRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "expires_at must be a valid RFC3339 timestamp", nil)
			return
		}
		if t.Before(time.Now()) {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "expires_at must be in the future", nil)
			return
		}
		expiresAt = &t
	}

	h.log.Debug("Creating conversation invite",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
	)

	invite, svcErr := h.inviteService.CreateInvite(ctx, convUUID, uuid.MustParse(userID), req.MaxUses, expiresAt)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Invite created successfully", invite)
}

// GetConversationInvites retrieves all invites for a conversation
func (h *ConversationHandler) GetConversationInvites(handler *request.RequestHandler) {
	ctx := handler.Context()

	_, ok := request.GetUserIDFromContext(ctx)
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

	invites, svcErr := h.inviteService.GetConversationInvites(ctx, convUUID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Invites retrieved",
		map[string]interface{}{
			"invites": invites,
		},
	)
}

// AcceptInvite accepts a conversation invite by code
func (h *ConversationHandler) AcceptInvite(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	inviteCode := handler.PathParam("code")
	if inviteCode == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invite code is required", nil)
		return
	}

	h.log.Debug("Accepting invite",
		logger.String("user_id", userID),
		logger.String("invite_code", inviteCode),
	)

	svcErr := h.inviteService.AcceptInvite(ctx, inviteCode, uuid.MustParse(userID))
	if svcErr != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Invite accepted successfully", nil)
}

// RevokeInvite revokes a conversation invite
func (h *ConversationHandler) RevokeInvite(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	inviteID := handler.PathParam("invite_id")
	if inviteID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invite ID is required", nil)
		return
	}
	invUUID, err := uuid.Parse(inviteID)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invite ID must be a valid UUID", nil)
		return
	}

	h.log.Debug("Revoking invite",
		logger.String("user_id", userID),
		logger.String("invite_id", inviteID),
	)

	svcErr := h.inviteService.RevokeInvite(ctx, invUUID, uuid.MustParse(userID))
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Invite revoked successfully", nil)
}
