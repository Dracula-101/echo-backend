package conversation

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// SetTypingIndicator handles POST /conversations/{id}/typing
func (h *ConversationHandler) SetTypingIndicator(handler *request.RequestHandler) {
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

	// Parse is_typing from query param or default to true for POST
	isTyping := handler.QueryParam("is_typing") != "false"

	h.log.Debug("Setting typing indicator",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
		logger.Bool("is_typing", isTyping),
	)

	svcErr := h.typingService.SetTyping(ctx, convUUID, uuid.MustParse(userID), isTyping)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Typing indicator updated", nil)
}

// GetTypingUsers handles GET /conversations/{id}/typing
func (h *ConversationHandler) GetTypingUsers(handler *request.RequestHandler) {
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

	users, svcErr := h.typingService.GetTypingUsers(ctx, convUUID, uuid.MustParse(userID))
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	type typingUserResp struct {
		UserID    string `json:"user_id"`
		StartedAt string `json:"started_at"`
	}
	resp := make([]typingUserResp, 0, len(users))
	for _, u := range users {
		resp = append(resp, typingUserResp{
			UserID:    u.UserID.String(),
			StartedAt: u.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Typing users retrieved",
		map[string]interface{}{
			"typing_users": resp,
		},
	)
}
