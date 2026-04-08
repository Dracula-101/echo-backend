package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	"net/http"
	"shared/server/request"
	"shared/server/response"
	"time"

	"github.com/google/uuid"
)

// CreatePoll creates a new poll in a conversation
func (h *MessageHandler) CreatePoll(handler *request.RequestHandler) {
	req := dto.NewCreatePollRequest()
	if !handler.ParseValidateAndSend(req) {
		return
	}

	userIDStr, ok := request.GetUserIDFromContext(handler.Context())
	if !ok {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}
	userID := uuid.MustParse(userIDStr)

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	input := domain.CreatePollInput{
		ConversationID: uuid.MustParse(req.ConversationID),
		CreatorUserID:  userID,
		Question:       req.Question,
		Options:        req.Options,
		AllowMultiple:  req.AllowMultiple,
		ExpiresAt:      expiresAt,
	}

	poll, options, appErr := h.pollService.CreatePoll(handler.Context(), input)
	if appErr != nil {
		h.sendAppError(handler, appErr)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), http.StatusCreated, "Poll created", map[string]interface{}{
		"poll":    poll,
		"options": options,
	})
}
