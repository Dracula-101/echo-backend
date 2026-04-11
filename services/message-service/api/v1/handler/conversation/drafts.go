package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// SaveDraft saves or updates a draft for a conversation
func (h *ConversationHandler) SaveDraft(handler *request.RequestHandler) {
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

	req := dto.NewUpsertDraftRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	var replyToID *uuid.UUID
	if req.ReplyToMessageID != nil {
		parsed := uuid.MustParse(*req.ReplyToMessageID)
		replyToID = &parsed
	}

	h.log.Debug("Saving draft",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
	)

	svcErr := h.draftService.UpsertDraft(ctx, convUUID, uuid.MustParse(userID), req.Content, replyToID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Draft saved successfully", nil)
}

// GetDraft retrieves a draft for a conversation
func (h *ConversationHandler) GetDraft(handler *request.RequestHandler) {
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

	draft, svcErr := h.draftService.GetDraft(ctx, convUUID, uuid.MustParse(userID))
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	if draft == nil {
		response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
			response.StatusOK, "No draft found", nil)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Draft retrieved", draft)
}

// DeleteDraft deletes a draft for a conversation
func (h *ConversationHandler) DeleteDraft(handler *request.RequestHandler) {
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

	svcErr := h.draftService.DeleteDraft(ctx, convUUID, uuid.MustParse(userID))
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Draft deleted successfully", nil)
}
