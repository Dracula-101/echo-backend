package conversation

import (
	"time"

	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// MuteConversation mutes notifications for a conversation
func (h *ConversationHandler) MuteConversation(handler *request.RequestHandler) {
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

	req := dto.NewMuteConversationRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	var muteUntil *time.Time
	if req.MuteUntil != nil {
		t, err := time.Parse(time.RFC3339, *req.MuteUntil)
		if err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "mute_until must be a valid RFC3339 timestamp", nil)
			return
		}
		if t.Before(time.Now()) {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "mute_until must be in the future", nil)
			return
		}
		muteUntil = &t
	}

	h.log.Debug("Muting conversation",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
	)

	svcErr := h.conversationService.MuteConversation(ctx, domain.MuteConversationInput{
		ConversationID: convUUID,
		UserID:         uuid.MustParse(userID),
		MuteUntil:      muteUntil,
	})
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Conversation muted successfully", nil)
}
