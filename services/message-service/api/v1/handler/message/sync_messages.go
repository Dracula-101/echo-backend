package message

import (
	"time"

	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) SyncMessages(handler *req.RequestHandler) {
	ctx := handler.Context()

	_, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	conversationID := handler.PathParam("conversation_id")
	if _, err := uuid.Parse(conversationID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", err)
		return
	}

	sinceStr := handler.QueryParam("since")
	if sinceStr == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "since is required", nil)
		return
	}
	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "since must be a valid RFC3339 timestamp", err)
		return
	}
	if since.Before(time.Now().Add(-30 * 24 * time.Hour)) {
		response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), 422, "Sync window exceeded",
			map[string]interface{}{"action": "full_reload"},
		)
		return
	}

	limit, qerr := handler.QueryParamInt("limit", 100)
	if qerr != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be an integer", nil)
		return
	}
	if limit < 1 || limit > 500 {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be between 1 and 500", nil)
		return
	}

	h.log.Debug("Syncing messages",
		logger.String("conversation_id", conversationID),
		logger.String("since", since.Format(time.RFC3339)),
		logger.Int("limit", limit),
	)

	messages, hasMore, msgErr := h.messageService.SyncMessages(ctx, conversationID, since, limit)
	if msgErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), msgErr.Message, msgErr.Error)
		return
	}

	messagesResponse := make([]dto.MessageResponse, 0, len(messages))
	var nextSince time.Time
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
		if msg.UpdatedAt.After(nextSince) {
			nextSince = msg.UpdatedAt
		}
	}
	if nextSince.IsZero() {
		nextSince = since
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Messages synced",
		map[string]interface{}{
			"messages":   messagesResponse,
			"has_more":   hasMore,
			"next_since": nextSince.Format(time.RFC3339Nano),
		},
	)
}
