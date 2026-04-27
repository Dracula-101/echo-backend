package message

import (
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

// GetMessages retrieves messages for a conversation with cursor-based pagination
//
// Flow:
//
//	[1] Parse and validate query parameters (conversation_id, limit, before/after cursor)
//	[2] Call MessageService.GetMessages or GetMessagesAfter with pagination params
//	[3] Return paginated response with messages and hasMore flag
//
// Pagination:
//   - Uses cursor-based pagination with before or after message ID
//   - "before" returns older messages in DESC order (scrolling back)
//   - "after" returns newer messages in ASC order (sync/catch-up)
//   - before and after are mutually exclusive
//   - Default limit: 20, max: 100
func (h *MessageHandler) GetMessages(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Get messages request received",
		logger.String("service", msgError.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	conversationID := handler.PathParam("conversation_id")
	if _, err := uuid.Parse(conversationID); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id must be a valid UUID", err)
		return
	}

	limit, err := handler.QueryParamInt("limit", 50)
	if err != nil {
		h.log.Warn("Invalid limit parameter",
			logger.String("service", msgError.ServiceName),
			logger.String("request_id", requestID),
			logger.Error(err),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be an integer", nil)
		return
	}
	if limit < 1 || limit > 100 {
		h.log.Warn("Invalid limit parameter",
			logger.String("service", msgError.ServiceName),
			logger.String("request_id", requestID),
			logger.Int("limit", limit),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "limit must be between 1 and 100", nil)
		return
	}

	beforeCursor := handler.QueryParam("before_id")
	afterCursor := handler.QueryParam("after_id")

	if beforeCursor != "" && afterCursor != "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "cannot specify both 'before_id' and 'after_id' cursors", nil)
		return
	}

	var beforeMessageID *string
	if beforeCursor != "" {
		if _, err := uuid.Parse(beforeCursor); err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "before_id must be a valid message ID (UUID)", err)
			return
		}
		beforeMessageID = &beforeCursor
	}

	if afterCursor != "" {
		if _, err := uuid.Parse(afterCursor); err != nil {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "after_id must be a valid message ID (UUID)", err)
			return
		}
	}

	h.log.Debug("Fetching messages",
		logger.String("service", msgError.ServiceName),
		logger.String("conversation_id", conversationID),
		logger.Int("limit", limit),
		logger.Any("before_message_id", beforeMessageID),
		logger.String("after_cursor", afterCursor),
	)

	// Fetch messages from service
	var messages []*domain.Message
	var hasMore bool
	var msgErr *msgError.MsgError

	if afterCursor != "" {
		messages, hasMore, msgErr = h.messageService.GetMessagesAfter(ctx, conversationID, afterCursor, limit)
	} else {
		messages, hasMore, msgErr = h.messageService.GetMessages(ctx, conversationID, limit, beforeMessageID)
	}
	if msgErr != nil {
		h.log.Error("Failed to fetch messages",
			logger.String("service", msgError.ServiceName),
			logger.String("request_id", requestID),
			logger.String("conversation_id", conversationID),
			logger.Error(msgErr.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), msgErr.Message, msgErr.Error)
		return
	}

	h.log.Info("Messages retrieved successfully",
		logger.String("service", msgError.ServiceName),
		logger.String("request_id", requestID),
		logger.String("conversation_id", conversationID),
		logger.Int("count", len(messages)),
		logger.Bool("has_more", hasMore),
	)

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
			CreatedAt:   msg.CreatedAt,
			UpdatedAt:   msg.UpdatedAt,
			EditedAt:    msg.EditedAt,
			DeletedAt:   msg.DeletedAt,
		})
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Messages retrieved successfully",
		dto.GetMessagesResponse{
			Messages: messagesResponse,
			HasMore:  hasMore,
			Limit:    limit,
		},
	)
}

// GetMessageByID retrieves a single message by its ID
func (h *MessageHandler) GetMessageByID(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Get message by ID request received",
		logger.String("service", msgError.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	// Get message ID from path parameter
	messageID := handler.PathParam("id")
	if messageID == "" {
		h.log.Warn("Missing message ID",
			logger.String("service", msgError.ServiceName),
			logger.String("request_id", requestID),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "message ID is required", nil)
		return
	}

	// Validate message ID format
	if _, err := uuid.Parse(messageID); err != nil {
		h.log.Warn("Invalid message ID format",
			logger.String("service", msgError.ServiceName),
			logger.String("request_id", requestID),
			logger.String("message_id", messageID),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "message ID must be a valid UUID", nil)
		return
	}

	h.log.Debug("Fetching message",
		logger.String("service", msgError.ServiceName),
		logger.String("message_id", messageID),
	)

	// Fetch message from service
	msg, msgErr := h.messageService.GetMessageByID(ctx, messageID)
	if msgErr != nil {
		h.log.Error("Failed to fetch message",
			logger.String("service", msgError.ServiceName),
			logger.String("request_id", requestID),
			logger.String("message_id", messageID),
			logger.Error(msgErr.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), msgErr.Message, msgErr.Error)
		return
	}

	h.log.Info("Message retrieved successfully",
		logger.String("service", msgError.ServiceName),
		logger.String("request_id", requestID),
		logger.String("message_id", messageID),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Message retrieved successfully",
		dto.MessageResponse{
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
		},
	)
}
