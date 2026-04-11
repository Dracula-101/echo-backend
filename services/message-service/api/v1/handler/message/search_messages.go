package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// SearchMessages searches for messages matching a query
func (h *MessageHandler) SearchMessages(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	request := dto.NewSearchMessagesRequest()
	if !handler.ParseValidateAndSend(request) {
		return
	}

	h.log.Debug("Searching messages",
		logger.String("user_id", userID),
		logger.String("query", request.Query),
	)

	messages, totalCount, err := h.messageService.SearchMessages(ctx, userID, request.Query, request.ConversationID, request.Limit, request.Offset)
	if err != nil {
		h.log.Error("Failed to search messages",
			logger.String("user_id", userID),
			logger.String("code", err.Code.String()),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		return
	}

	// Build response
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

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Search results",
		map[string]interface{}{
			"messages":    messagesResponse,
			"total_count": totalCount,
			"limit":       request.Limit,
			"offset":      request.Offset,
		},
	)
}
