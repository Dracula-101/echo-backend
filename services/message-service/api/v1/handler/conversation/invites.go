package conversation

import (
	"time"

	"echo-backend/services/message-service/api/v1/dto"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) CreateInvite(handler *request.RequestHandler) {
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

	req := dto.NewCreateInviteRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "expires_at must be a valid RFC3339 timestamp", err)
			return
		}
		if t.Before(time.Now()) {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "expires_at must be in the future", nil)
			return
		}
		expiresAt = &t
	}

	h.log.Debug("Creating conversation invite",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
	)

	invite, svcErr := h.inviteService.CreateInvite(ctx, convUUID, userID, req.MaxUses, expiresAt)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusCreated, "Invite created", invite)
}

func (h *ConversationHandler) GetConversationInvites(handler *request.RequestHandler) {
	ctx := handler.Context()

	if _, ok := h.authedUserID(handler); !ok {
		return
	}

	convUUID, err := uuid.Parse(handler.PathParam("id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", err)
		return
	}

	invites, svcErr := h.inviteService.GetConversationInvites(ctx, convUUID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Invites retrieved",
		map[string]interface{}{"invites": invites},
	)
}

func (h *ConversationHandler) AcceptInvite(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	inviteCode := handler.PathParam("code")
	if inviteCode == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invite code is required", nil)
		return
	}

	h.log.Debug("Accepting invite",
		logger.String("user_id", userID.String()),
		logger.String("invite_code", inviteCode),
	)

	if svcErr := h.inviteService.AcceptInvite(ctx, inviteCode, userID); svcErr != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Invite accepted", nil)
}

func (h *ConversationHandler) RevokeInvite(handler *request.RequestHandler) {
	ctx := handler.Context()

	userID, ok := h.authedUserID(handler)
	if !ok {
		return
	}

	invUUID, err := uuid.Parse(handler.PathParam("invite_id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invite ID must be a valid UUID", err)
		return
	}

	h.log.Debug("Revoking invite",
		logger.String("user_id", userID.String()),
		logger.String("invite_id", invUUID.String()),
	)

	if svcErr := h.inviteService.RevokeInvite(ctx, invUUID, userID); svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
