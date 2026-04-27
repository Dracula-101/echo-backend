package service

import (
	"context"
	"encoding/json"
	"time"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	internalMsg "echo-backend/services/message-service/internal/messaging"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/common/hashing"
	"shared/server/request"

	"github.com/google/uuid"
)

// EditMessage handles editing an existing message
func (s *messageService) EditMessage(ctx context.Context, messageID string, userID string, newContent string) (*domain.Message, *msgError.MsgError) {
	// Fetch message
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return nil, &msgError.MsgError{
			Message: "Message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	msg := domain.NewMessage(*dbMsg)

	// Check if user can edit
	parsedUserID := uuid.MustParse(userID)
	if msg.IsDeleted {
		return nil, &msgError.MsgError{
			Message: "Cannot edit a deleted message",
			Code:    msgError.CodeMessageNotEditable,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "message is deleted"),
		}
	}
	if msg.SenderUserID != parsedUserID {
		return nil, &msgError.MsgError{
			Message: "Only the sender can edit this message",
			Code:    msgError.CodeMessageNotEditable,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not the sender"),
		}
	}
	editDeadline := msg.CreatedAt.Add(24 * time.Hour)
	if time.Now().After(editDeadline) {
		return nil, &msgError.MsgError{
			Message: "Edit window has expired (24 hours)",
			Code:    msgError.CodeMessageEditExpired,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "edit window expired"),
		}
	}

	// Update content
	if err := s.messageRepo.UpdateMessageContent(ctx, messageID, newContent); err != nil {
		s.logger.Error("Failed to update message content",
			logger.String("service", msgError.ServiceName),
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to edit message",
			Code:    msgError.CodeMessageEditFailed,
			Error:   err,
		}
	}

	// Publish event
	s.eventPublisher.PublishMessageEvent(ctx, &domain.MessageEvent{
		EventType:      domain.MessageEventTypeEdited,
		MessageID:      msg.ID,
		ConversationID: msg.ConversationID,
		SenderUserID:   msg.SenderUserID,
		Timestamp:      time.Now(),
		Payload:        map[string]interface{}{"new_content": newContent},
	})

	// Return updated message
	msg.Content = newContent
	msg.IsEdited = true
	now := time.Now()
	msg.EditedAt = &now
	return &msg, nil
}

// DeleteMessage handles soft-deleting a message
func (s *messageService) DeleteMessage(ctx context.Context, messageID string, userID string, deleteForEveryone bool) *msgError.MsgError {
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return &msgError.MsgError{
			Message: "Message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	msg := domain.NewMessage(*dbMsg)
	parsedUserID := uuid.MustParse(userID)

	if msg.IsDeleted {
		return nil // Idempotent
	}

	// Check permissions: sender can always delete, admins can delete for everyone
	if msg.SenderUserID != parsedUserID {
		// Check if user is admin/owner in the conversation
		participants, _ := s.conversationRepo.GetConversationParticipants(ctx, msg.ConversationID)
		hasPermission := false
		for _, p := range participants {
			if p.UserID == parsedUserID && p.CanDeleteMessages {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return &msgError.MsgError{
				Message: "You don't have permission to delete this message",
				Code:    msgError.CodeMessageNotDeletable,
				Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized to delete"),
			}
		}
	}

	if err := s.messageRepo.SoftDeleteMessage(ctx, messageID, deleteForEveryone); err != nil {
		s.logger.Error("Failed to delete message",
			logger.String("service", msgError.ServiceName),
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to delete message",
			Code:    msgError.CodeMessageDeleteFailed,
			Error:   err,
		}
	}

	// Publish event
	s.eventPublisher.PublishMessageEvent(ctx, &domain.MessageEvent{
		EventType:      domain.MessageEventTypeDeleted,
		MessageID:      msg.ID,
		ConversationID: msg.ConversationID,
		SenderUserID:   parsedUserID,
		Timestamp:      time.Now(),
	})

	return nil
}

// ForwardMessage forwards a message to another conversation
func (s *messageService) ForwardMessage(ctx context.Context, messageID string, userID string, targetConversationID string, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MsgError) {
	// Fetch original message
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return nil, &msgError.MsgError{
			Message: "Original message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	originalMsg := domain.NewMessage(*dbMsg)
	if originalMsg.IsDeleted {
		return nil, &msgError.MsgError{
			Message: "Cannot forward a deleted message",
			Code:    msgError.CodeMessageNotFound,
			Error:   pkgErrors.New(pkgErrors.CodeNotFound, "message is deleted"),
		}
	}

	parsedUserID := uuid.MustParse(userID)
	parsedTargetID := uuid.MustParse(targetConversationID)

	// Validate user is participant in target conversation
	targetErr := s.conversationRepo.ValidateConversationParticipant(parsedTargetID, parsedUserID)
	if targetErr != nil {
		return nil, &msgError.MsgError{
			Message: "You are not a participant in the target conversation",
			Code:    msgError.CodeTargetConversationNotFound,
			Error:   targetErr,
		}
	}

	// Get target conversation participants for delivery tracking
	participants, _ := s.conversationRepo.GetConversationParticipants(ctx, parsedTargetID)
	var recipientIDs []uuid.UUID
	for _, p := range participants {
		if p.UserID != parsedUserID {
			recipientIDs = append(recipientIDs, p.UserID)
		}
	}

	// Build forwarded message event
	forwardedFromID := messageID
	content := utils.DerefString(dbMsg.Content)

	hashSvc, _ := hashing.DefaultService()
	messageEvent := domain.ChatMessageEvent{
		ID:                     uuid.New(),
		ConversationID:         parsedTargetID,
		SenderUserID:           parsedUserID,
		RecipientIDs:           recipientIDs,
		MessageType:            originalMsg.MessageType,
		Content:                content,
		ContentEncrypted:       false,
		ContentHash:            hashSvc.HashString(content),
		FormatType:             domain.MessageFormatPlain,
		Status:                 domain.MessageStatusSent,
		Mentions:               json.RawMessage("[]"),
		Links:                  json.RawMessage("[]"),
		Hashtags:               []string{},
		IsForwarded:            true,
		ForwardedFromMessageID: &forwardedFromID,
		SentFromDeviceID:       deviceInfo.ID,
		SentFromIP:             clientIP,
		Metadata:               dbMsg.Metadata,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	publishErr := s.eventPublisher.PublishChatMessage(ctx, &messageEvent)
	if publishErr != nil {
		return nil, &msgError.MsgError{
			Message: "Failed to forward message",
			Code:    msgError.CodeForwardFailed,
			Error:   pkgErrors.FromError(publishErr, pkgErrors.CodeInternal, "failed to publish forwarded message"),
		}
	}

	// Publish forward event on original message
	s.eventPublisher.PublishMessageEvent(ctx, &domain.MessageEvent{
		EventType:      domain.MessageEventTypeForwarded,
		MessageID:      originalMsg.ID,
		ConversationID: originalMsg.ConversationID,
		SenderUserID:   parsedUserID,
		Timestamp:      time.Now(),
	})

	msg := &domain.Message{
		ID:             messageEvent.ID,
		ConversationID: parsedTargetID,
		SenderUserID:   parsedUserID,
		MessageType:    originalMsg.MessageType,
		Content:        content,
		Status:         domain.MessageStatusSent,
		CreatedAt:      messageEvent.CreatedAt,
		UpdatedAt:      messageEvent.UpdatedAt,
		Metadata:       originalMsg.Metadata,
	}
	return msg, nil
}

// PinMessage pins a message
func (s *messageService) PinMessage(ctx context.Context, messageID string, userID string) *msgError.MsgError {
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return &msgError.MsgError{
			Message: "Message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	msg := domain.NewMessage(*dbMsg)
	parsedUserID := uuid.MustParse(userID)

	// Validate user is participant with permissions
	participants, _ := s.conversationRepo.GetConversationParticipants(ctx, msg.ConversationID)
	hasPermission := false
	for _, p := range participants {
		if p.UserID == parsedUserID && p.HasPermissionToManage() {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return &msgError.MsgError{
			Message: "You don't have permission to pin messages",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized to pin"),
		}
	}

	if err := s.messageRepo.PinMessage(ctx, msg.ConversationID.String(), messageID, userID); err != nil {
		if err.Code() == pkgErrors.CodeAlreadyExists {
			return &msgError.MsgError{
				Message: "Message is already pinned",
				Code:    msgError.CodeAlreadyPinned,
				Error:   err,
			}
		}
		return &msgError.MsgError{
			Message: "Failed to pin message",
			Code:    msgError.CodePinFailed,
			Error:   err,
		}
	}

	s.eventPublisher.PublishMessageEvent(ctx, &domain.MessageEvent{
		EventType:      domain.MessageEventTypePinned,
		MessageID:      msg.ID,
		ConversationID: msg.ConversationID,
		SenderUserID:   parsedUserID,
		Timestamp:      time.Now(),
	})

	return nil
}

// UnpinMessage unpins a message
func (s *messageService) UnpinMessage(ctx context.Context, messageID string, userID string) *msgError.MsgError {
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return &msgError.MsgError{
			Message: "Message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	msg := domain.NewMessage(*dbMsg)
	parsedUserID := uuid.MustParse(userID)

	// Validate user has manage permissions
	participants, _ := s.conversationRepo.GetConversationParticipants(ctx, msg.ConversationID)
	hasPermission := false
	for _, p := range participants {
		if p.UserID == parsedUserID && p.HasPermissionToManage() {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return &msgError.MsgError{
			Message: "You don't have permission to unpin messages",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized to unpin"),
		}
	}

	if err := s.messageRepo.UnpinMessage(ctx, msg.ConversationID.String(), messageID); err != nil {
		if err.Code() == pkgErrors.CodeNotFound {
			return &msgError.MsgError{
				Message: "Message is not pinned",
				Code:    msgError.CodeNotPinned,
				Error:   err,
			}
		}
		return &msgError.MsgError{
			Message: "Failed to unpin message",
			Code:    msgError.CodePinFailed,
			Error:   err,
		}
	}

	s.eventPublisher.PublishMessageEvent(ctx, &domain.MessageEvent{
		EventType:      domain.MessageEventTypeUnpinned,
		MessageID:      msg.ID,
		ConversationID: msg.ConversationID,
		SenderUserID:   parsedUserID,
		Timestamp:      time.Now(),
	})

	return nil
}

// GetPinnedMessages retrieves pinned messages for a conversation
func (s *messageService) GetPinnedMessages(ctx context.Context, conversationID string) ([]*domain.Message, *msgError.MsgError) {
	dbMessages, dbErr := s.messageRepo.GetPinnedMessages(ctx, conversationID)
	if dbErr != nil {
		return nil, &msgError.MsgError{
			Message: "Failed to get pinned messages",
			Code:    msgError.CodeMessageRetrievalFailed,
			Error:   dbErr,
		}
	}

	messages := make([]*domain.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg := domain.NewMessage(*dbMsg)
		messages = append(messages, &msg)
	}
	return messages, nil
}

// MarkAsRead marks messages up to a given message as read
func (s *messageService) MarkAsRead(ctx context.Context, messageID string, userID string) *msgError.MsgError {
	// Fetch message to get conversation_id
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return &msgError.MsgError{
			Message: "Message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	parsedUserID := uuid.MustParse(userID)
	conversationID := uuid.MustParse(dbMsg.ConversationID)

	// Validate participant
	valErr := s.conversationRepo.ValidateConversationParticipant(conversationID, parsedUserID)
	if valErr != nil {
		return &msgError.MsgError{
			Message: "You are not a participant in this conversation",
			Code:    msgError.CodeParticipantNotInConversation,
			Error:   valErr,
		}
	}

	// Mark as read
	if err := s.deliveryRepo.MarkConversationAsRead(ctx, userID, dbMsg.ConversationID, messageID); err != nil {
		s.logger.Error("Failed to mark as read",
			logger.String("service", msgError.ServiceName),
			logger.String("message_id", messageID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to mark messages as read",
			Code:    msgError.CodeMarkAsReadFailed,
			Error:   err,
		}
	}

	// Publish delivery event
	msgID := uuid.MustParse(messageID)
	s.eventPublisher.PublishDeliveryEvent(ctx, &internalMsg.DeliveryEvent{
		EventType:      internalMsg.DeliveryEventRead,
		MessageIDs:     []uuid.UUID{msgID},
		UserID:         parsedUserID,
		ConversationID: conversationID,
		Timestamp:      time.Now(),
	})

	return nil
}

// GetDeliveryStatus returns delivery status summary for a message
func (s *messageService) GetDeliveryStatus(ctx context.Context, messageID string, userID string) (*domain.DeliveryStatusSummary, *msgError.MsgError) {
	// Fetch message
	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		return nil, &msgError.MsgError{
			Message: "Message not found",
			Code:    msgError.CodeMessageNotFound,
			Error:   dbErr,
		}
	}

	// Only the sender should see detailed delivery status
	if dbMsg.SenderUserID != userID {
		return nil, &msgError.MsgError{
			Message: "Only the sender can view delivery status",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not the sender"),
		}
	}

	statuses, err := s.deliveryRepo.GetDeliveryStatuses(ctx, messageID)
	if err != nil {
		return nil, &msgError.MsgError{
			Message: "Failed to get delivery status",
			Code:    msgError.CodeDeliveryStatusFailed,
			Error:   err,
		}
	}

	msgID := uuid.MustParse(messageID)
	summary := &domain.DeliveryStatusSummary{
		MessageID:       msgID,
		TotalRecipients: len(statuses),
	}

	for _, s := range statuses {
		switch s.Status {
		case "read":
			summary.ReadCount++
			if s.ReadAt != nil {
				summary.ReadBy = append(summary.ReadBy, domain.ReadReceipt{
					UserID: uuid.MustParse(s.UserID),
					ReadAt: *s.ReadAt,
				})
			}
		case "delivered":
			summary.DeliveredCount++
			if s.DeliveredAt != nil {
				summary.DeliveredBy = append(summary.DeliveredBy, domain.DeliveryReceipt{
					UserID:      uuid.MustParse(s.UserID),
					DeliveredAt: *s.DeliveredAt,
				})
			}
		case "sent":
			summary.PendingCount++
		case "failed":
			summary.FailedCount++
		}
	}

	return summary, nil
}

func (s *messageService) SearchMessages(ctx context.Context, userID string, query string, conversationID *string, limit int, cursor *time.Time) ([]*domain.Message, int, *msgError.MsgError) {
	dbMessages, totalCount, dbErr := s.messageRepo.SearchMessages(ctx, query, userID, conversationID, limit, cursor)
	if dbErr != nil {
		return nil, 0, &msgError.MsgError{
			Message: "Failed to search messages",
			Code:    msgError.CodeSearchFailed,
			Error:   dbErr,
		}
	}

	messages := make([]*domain.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg := domain.NewMessage(*dbMsg)
		messages = append(messages, &msg)
	}
	return messages, totalCount, nil
}

func (s *messageService) SyncMessages(ctx context.Context, conversationID string, since time.Time, limit int) ([]*domain.Message, bool, *msgError.MsgError) {
	dbMessages, dbErr := s.messageRepo.GetMessagesUpdatedSince(ctx, conversationID, since, limit+1)
	if dbErr != nil {
		return nil, false, &msgError.MsgError{
			Message: "Failed to sync messages",
			Code:    msgError.CodeMessageRetrievalFailed,
			Error:   dbErr,
		}
	}

	hasMore := len(dbMessages) > limit
	if hasMore {
		dbMessages = dbMessages[:limit]
	}

	messages := make([]*domain.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg := domain.NewMessage(*dbMsg)
		messages = append(messages, &msg)
	}
	return messages, hasMore, nil
}

// GetThread retrieves thread/reply messages for a parent message
func (s *messageService) GetThread(ctx context.Context, parentMessageID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MsgError) {
	// Fetch limit+1 to determine hasMore
	dbMessages, dbErr := s.messageRepo.GetThreadMessages(ctx, parentMessageID, limit+1, beforeMessageID)
	if dbErr != nil {
		return nil, false, &msgError.MsgError{
			Message: "Failed to get thread messages",
			Code:    msgError.CodeMessageRetrievalFailed,
			Error:   dbErr,
		}
	}

	hasMore := len(dbMessages) > limit
	if hasMore {
		dbMessages = dbMessages[:limit]
	}

	messages := make([]*domain.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg := domain.NewMessage(*dbMsg)
		messages = append(messages, &msg)
	}
	return messages, hasMore, nil
}
