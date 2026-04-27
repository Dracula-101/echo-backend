package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/repo"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) GetSettings(handler *request.RequestHandler) {
	ctx := handler.Context()

	if _, ok := h.authedUserID(handler); !ok {
		return
	}

	convUUID, err := uuid.Parse(handler.PathParam("id"))
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", err)
		return
	}

	settings, svcErr := h.settingsService.GetSettings(ctx, convUUID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	if settings == nil {
		response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
			response.StatusOK, "No settings found", nil)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Settings retrieved", settings)
}

func (h *ConversationHandler) UpdateSettings(handler *request.RequestHandler) {
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

	req := dto.NewUpdateSettingsRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	h.log.Debug("Updating conversation settings",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
	)

	if svcErr := h.settingsService.UpdateSettings(ctx, convUUID, userID, repo.ConversationSettingsInput{
		DisappearingMessagesEnabled:  req.DisappearingMessagesEnabled,
		DisappearingMessagesDuration: req.DisappearingMessagesDuration,
		ReadReceiptsEnabled:          req.ReadReceiptsEnabled,
		TypingIndicatorsEnabled:      req.TypingIndicatorsEnabled,
		LinkPreviewsEnabled:          req.LinkPreviewsEnabled,
		AutoDownloadMedia:            req.AutoDownloadMedia,
	}); svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Settings updated", nil)
}
