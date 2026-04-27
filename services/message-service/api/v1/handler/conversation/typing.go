package conversation

import (
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *ConversationHandler) SetTypingIndicator(handler *request.RequestHandler) {
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

	isTyping := handler.QueryParam("is_typing") != "false"

	h.log.Debug("Setting typing indicator",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", convUUID.String()),
		logger.Bool("is_typing", isTyping),
	)

	if svcErr := h.typingService.SetTyping(ctx, convUUID, userID, isTyping); svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
		return
	}

	handler.Writer().WriteHeader(response.StatusNoContent)
}

func (h *ConversationHandler) GetTypingUsers(handler *request.RequestHandler) {
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

	users, svcErr := h.typingService.GetTypingUsers(ctx, convUUID, userID)
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
		map[string]interface{}{"typing_users": resp},
	)
}
