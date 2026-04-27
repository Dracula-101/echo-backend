package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) SaveDraft(handler *request.RequestHandler) {
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

	req := dto.NewUpsertDraftRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	var replyToID *uuid.UUID
	if req.ReplyToMessageID != nil {
		parsed, perr := uuid.Parse(*req.ReplyToMessageID)
		if perr != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "reply_to_message_id must be a valid UUID", perr)
			return
		}
		replyToID = &parsed
	}

	h.log.Debug("Saving draft",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
	)

	if svcErr := h.draftService.UpsertDraft(ctx, convUUID, userID, req.Content, replyToID); svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Draft saved", nil)
}

func (h *ConversationHandler) GetDraft(handler *request.RequestHandler) {
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

	draft, svcErr := h.draftService.GetDraft(ctx, convUUID, userID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	if draft == nil {
		response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
			response.StatusOK, "No draft", nil)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Draft retrieved", draft)
}

func (h *ConversationHandler) DeleteDraft(handler *request.RequestHandler) {
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

	if svcErr := h.draftService.DeleteDraft(ctx, convUUID, userID); svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}
