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
	userID, perr := uuid.Parse(userIDStr)
	if perr != nil {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Invalid user id", perr)
		return
	}

	var closesAt *time.Time
	if req.ClosesAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ClosesAt)
		if err == nil {
			closesAt = &t
		}
	}

	convID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", err)
		return
	}

	input := domain.CreatePollInput{
		ConversationID:     convID,
		CreatorUserID:      userID,
		Question:           req.Question,
		Options:            req.Options,
		AllowMultiple:      req.AllowMultipleAnswers,
		IsAnonymous:        req.IsAnonymous,
		IsQuiz:             req.IsQuiz,
		CorrectOptionIndex: req.CorrectOptionIndex,
		Explanation:        req.Explanation,
		ExpiresAt:          closesAt,
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
