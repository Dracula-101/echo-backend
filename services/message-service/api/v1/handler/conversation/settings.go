package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetSettings retrieves conversation settings
func (h *ConversationHandler) GetSettings(handler *request.RequestHandler) {
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

// UpdateSettings updates conversation settings
func (h *ConversationHandler) UpdateSettings(handler *request.RequestHandler) {
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

	req := dto.NewUpdateSettingsRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	h.log.Debug("Updating conversation settings",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
	)

	svcErr := h.settingsService.UpdateSettings(ctx, convUUID, uuid.MustParse(userID), repo.ConversationSettingsInput{
		DisappearingMessagesEnabled:  req.DisappearingMessagesEnabled,
		DisappearingMessagesDuration: req.DisappearingMessagesDuration,
		ReadReceiptsEnabled:          req.ReadReceiptsEnabled,
		TypingIndicatorsEnabled:      req.TypingIndicatorsEnabled,
		LinkPreviewsEnabled:          req.LinkPreviewsEnabled,
		AutoDownloadMedia:            req.AutoDownloadMedia,
	})
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Settings updated successfully", nil)
}
