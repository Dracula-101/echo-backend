package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetThread retrieves replies/thread messages for a parent message
func (h *MessageHandler) GetThread(handler *req.RequestHandler) {
	ctx := handler.Context()

	_, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", nil)
		return
	}

	parentMessageID := handler.PathParam("id")
	if parentMessageID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID is required", nil)
		return
	}
	if _, err := uuid.Parse(parentMessageID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Message ID must be a valid UUID", nil)
		return
	}

	limit, err := handler.QueryParamInt("limit", 50)
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be an integer", nil)
		return
	}
	if limit < 1 || limit > 100 {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be between 1 and 100", nil)
		return
	}

	var beforeID *string
	if before := handler.QueryParam("before_id"); before != "" {
		if _, err := uuid.Parse(before); err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "before_id must be a valid UUID", err)
			return
		}
		beforeID = &before
	}

	h.log.Debug("Getting thread messages",
		logger.String("parent_message_id", parentMessageID),
		logger.Int("limit", limit),
	)

	parent, parentErr := h.messageService.GetMessageByID(ctx, parentMessageID)
	if parentErr != nil {
		response.NotFoundError(ctx, handler.Request(), handler.Writer(), "Parent message not found")
		return
	}

	messages, hasMore, msgErr := h.messageService.GetThread(ctx, parentMessageID, limit, beforeID)
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
			CreatedAt:   msg.CreatedAt,
			UpdatedAt:   msg.UpdatedAt,
			EditedAt:    msg.EditedAt,
			DeletedAt:   msg.DeletedAt,
		})
	}

	parentResp := dto.MessageResponse{
		ID:             parent.ID.String(),
		ConversationID: parent.ConversationID.String(),
		SenderUserID:   parent.SenderUserID.String(),
		Content:        parent.Content,
		MessageType:    string(parent.MessageType),
		Status:         string(parent.Status),
		IsEdited:       parent.IsEdited,
		IsDeleted:      parent.IsDeleted,
		ReadCount:      parent.ReadCount,
		Metadata:       parent.Metadata,
		CreatedAt:      parent.CreatedAt,
		UpdatedAt:      parent.UpdatedAt,
		EditedAt:       parent.EditedAt,
		DeletedAt:      parent.DeletedAt,
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Thread messages retrieved",
		map[string]interface{}{
			"parent_message": parentResp,
			"replies":        messagesResponse,
			"has_more":       hasMore,
		},
	)
}
