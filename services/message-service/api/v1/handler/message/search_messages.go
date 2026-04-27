package message

import (
	"time"

	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

func (h *MessageHandler) SearchMessages(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	conversationID := handler.PathParam("conversation_id")
	if _, err := uuid.Parse(conversationID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", err)
		return
	}

	request := dto.NewSearchMessagesRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	var cursor *time.Time
	if request.Cursor != "" {
		t, perr := time.Parse(time.RFC3339, request.Cursor)
		if perr != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "cursor must be a valid RFC3339 timestamp", perr)
			return
		}
		cursor = &t
	}

	h.log.Debug("Searching messages",
		logger.String("user_id", userID),
		logger.String("conversation_id", conversationID),
		logger.String("q", request.Query),
	)

	messages, totalCount, err := h.messageService.SearchMessages(ctx, userID, request.Query, &conversationID, request.Limit, cursor)
	if err != nil {
		h.log.Error("Failed to search messages",
			logger.String("user_id", userID),
			logger.String("code", err.Code.String()),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
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

	var nextCursor *string
	if len(messagesResponse) == request.Limit && len(messagesResponse) > 0 {
		c := messages[len(messages)-1].CreatedAt.Format(time.RFC3339Nano)
		nextCursor = &c
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Search results",
		map[string]interface{}{
			"messages":    messagesResponse,
			"total_count": totalCount,
			"limit":       request.Limit,
			"next_cursor": nextCursor,
		},
	)
}
