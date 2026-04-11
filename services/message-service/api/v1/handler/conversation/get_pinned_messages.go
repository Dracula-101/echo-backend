package conversation

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetPinnedMessages retrieves all pinned messages in a conversation
func (h *ConversationHandler) GetPinnedMessages(handler *request.RequestHandler) {
	ctx := handler.Context()

	_, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	conversationID := handler.PathParam("id")
	if conversationID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID is required", nil)
		return
	}
	if _, err := uuid.Parse(conversationID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Conversation ID must be a valid UUID", nil)
		return
	}

	h.log.Debug("Getting pinned messages",
		logger.String("conversation_id", conversationID),
	)

	messages, svcErr := h.messageService.GetPinnedMessages(ctx, conversationID)
	if svcErr != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), svcErr.Message, svcErr.Error)
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

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(),
		response.StatusOK, "Pinned messages retrieved",
		map[string]interface{}{
			"messages": messagesResponse,
		},
	)
}
