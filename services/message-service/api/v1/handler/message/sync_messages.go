package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// SyncMessages retrieves messages after a given message ID for client sync/catch-up
func (h *MessageHandler) SyncMessages(handler *req.RequestHandler) {
	ctx := handler.Context()

	_, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	conversationID := handler.QueryParam("conversation_id")
	if conversationID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id is required", nil)
		return
	}
	if _, err := uuid.Parse(conversationID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", nil)
		return
	}

	lastMessageID := handler.QueryParam("last_message_id")
	if lastMessageID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "last_message_id is required", nil)
		return
	}
	if _, err := uuid.Parse(lastMessageID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "last_message_id must be a valid UUID", nil)
		return
	}

	limit, err := handler.QueryParamInt("limit", 100)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be an integer", nil)
		return
	}
	if limit < 1 || limit > 500 {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be between 1 and 500", nil)
		return
	}

	h.log.Debug("Syncing messages",
		logger.String("conversation_id", conversationID),
		logger.String("last_message_id", lastMessageID),
		logger.Int("limit", limit),
	)

	messages, hasMore, msgErr := h.messageService.SyncMessages(ctx, conversationID, lastMessageID, limit)
	if msgErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), msgErr.Message, msgErr.Error)
		return
	}

	messagesResponse := make([]dto.MessageResponse, 0, len(messages))
	for _, msg := range messages {
		messagesResponse = append(messagesResponse, dto.MessageResponse{
			ID:             msg.ID.String(),
			ConversationID: msg.ConversationID.String(),
			SenderUserID:   msg.SenderUserID.String(),
			ParentMessageID: func() *string {
				if msg.ParentMessageID != nil {
					s := msg.ParentMessageID.String()
					return &s
				}
				return nil
			}(),
			Content:     msg.Content,
			MessageType: string(msg.MessageType),
			Status:      string(msg.Status),
			IsEdited:    msg.IsEdited,
			IsDeleted:   msg.IsDeleted,
			ReadCount:   msg.ReadCount,
			Metadata:    msg.Metadata,
			CreatedAt:   msg.CreatedAt,
			UpdatedAt:   msg.UpdatedAt,
			EditedAt:    msg.EditedAt,
			DeletedAt:   msg.DeletedAt,
		})
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Messages synced",
		map[string]interface{}{
			"messages": messagesResponse,
			"has_more": hasMore,
		},
	)
}
