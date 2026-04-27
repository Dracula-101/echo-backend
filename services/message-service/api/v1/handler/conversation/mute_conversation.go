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

func (h *ConversationHandler) MuteConversation(handler *request.RequestHandler) {
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

	req := dto.NewMuteConversationRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	var muteUntil *time.Time
	if req.MuteUntil != nil {
		t, err := time.Parse(time.RFC3339, *req.MuteUntil)
		if err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "mute_until must be a valid RFC3339 timestamp", err)
			return
		}
		if t.Before(time.Now()) {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "mute_until must be in the future", nil)
			return
		}
		muteUntil = &t
	}

	h.log.Debug("Muting conversation",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
	)

	svcErr := h.conversationService.MuteConversation(ctx, domain.MuteConversationInput{
		ConversationID: convUUID,
		UserID:         userID,
		MuteUntil:      muteUntil,
	})
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Conversation muted",
		map[string]interface{}{
			"is_muted":    true,
			"muted_until": muteUntil,
		},
	)
}
