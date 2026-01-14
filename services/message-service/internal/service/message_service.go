package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"echo-backend/services/message-service/internal/domain"
	"echo-backend/services/message-service/internal/models"
	"echo-backend/services/message-service/internal/repo"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/messaging"

	"github.com/google/uuid"
)

type MessageService interface {
	SendMessage(ctx context.Context, req *models.SendMessageRequest) (*models.Message, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, params *models.PaginationParams) (*models.MessagesResponse, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*models.Message, error)
	EditMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, newContent string) error
	DeleteMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error
	MarkAsDelivered(ctx context.Context, messageID, userID uuid.UUID) error
	MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) error
	HandleReadReceipt(ctx context.Context, userID, messageID uuid.UUID) error
	SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) error
	MarkConversationAsRead(ctx context.Context, conversationID, userID uuid.UUID) error
}

type messageService struct {
	repo           repo.MessageRepository
	eventPublisher EventPublisher
	kafka          messaging.Producer
	logger         logger.Logger
}

func NewMessageService(
	repo repo.MessageRepository,
	eventPublisher EventPublisher,
	kafka messaging.Producer,
	log logger.Logger,
) MessageService {
	return &messageService{
		repo:           repo,
		eventPublisher: eventPublisher,
		kafka:          kafka,
		logger:         log,
	}
}

// SendMessage handles the complete flow of sending a message
func (s *messageService) SendMessage(ctx context.Context, req *models.SendMessageRequest) (*models.Message, error) {
	var canSend bool
	var err error
	canSend, err = s.repo.ValidateParticipant(ctx, req.ConversationID, req.SenderUserID)

	if err != nil {
		if appErr, ok := err.(pkgErrors.AppError); ok {
			return nil, appErr.WithService("message-service")
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to validate participant").
			WithService("message-service").
			WithDetail("conversation_id", req.ConversationID.String()).
			WithDetail("user_id", req.SenderUserID.String())
	}

	if !canSend {
		s.logger.Warn("User not authorized to send message",
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("user_id", req.SenderUserID.String()),
		)
		return nil, pkgErrors.New(pkgErrors.CodeUnauthorized, "user not authorized to send messages in this conversation").
			WithService("message-service").
			WithDetail("conversation_id", req.ConversationID.String()).
			WithDetail("user_id", req.SenderUserID.String())
	}

	now := time.Now()
	message := &models.Message{
		ID:              uuid.New(),
		ConversationID:  req.ConversationID,
		SenderUserID:    req.SenderUserID,
		ParentMessageID: req.ParentMessageID,
		Content:         req.Content,
		MessageType:     req.MessageType,
		Status:          "sent",
		IsEdited:        false,
		IsDeleted:       false,
		UpdatedAt:       now,
	}

	// Always set valid JSON for mentions (empty array if no mentions)
	if len(req.Mentions) > 0 {
		mentionsJSON, err := json.Marshal(req.Mentions)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to marshal mentions").
				WithService("message-service").
				WithDetail("message_id", message.ID.String())
		}
		message.Mentions = mentionsJSON
	} else {
		message.Mentions = json.RawMessage("[]") // Empty array
	}

	// Always set valid JSON for metadata (empty object if no metadata)
	if req.Metadata != (models.Metadata{}) {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to marshal metadata").
				WithService("message-service").
				WithDetail("message_id", message.ID.String())
		}
		message.Metadata = metadataJSON
	} else {
		message.Metadata = json.RawMessage("{}") // Empty object
	}

	err = s.repo.CreateMessage(ctx, message)

	if err != nil {
		if appErr, ok := err.(pkgErrors.AppError); ok {
			return nil, appErr.WithService("message-service")
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create message").
			WithService("message-service").
			WithDetail("message_id", message.ID.String())
	}

	s.logger.Info("Message created",
		logger.String("message_id", message.ID.String()),
		logger.String("conversation_id", message.ConversationID.String()),
		logger.String("sender_id", message.SenderUserID.String()),
	)

	var participantIDs []uuid.UUID
	participantIDs, err = s.repo.GetParticipantUserIDs(ctx, req.ConversationID)

	if err != nil {
		s.logger.Error("Failed to get participants",
			logger.String("conversation_id", req.ConversationID.String()),
			logger.Error(err),
		)
		participantIDs = []uuid.UUID{}
	}

	go func() {
		bgCtx := context.Background()
		s.repo.UpdateConversationLastMessage(bgCtx, req.ConversationID, message.ID)
	}()

	recipientIDs := make([]uuid.UUID, 0)
	for _, participantID := range participantIDs {
		if participantID != req.SenderUserID {
			recipientIDs = append(recipientIDs, participantID)
		}
	}

	if len(recipientIDs) > 0 {
		go func() {
			bgCtx := context.Background()
			s.repo.CreateDeliveryStatus(bgCtx, message.ID, recipientIDs)
		}()
	}

	// Step 7: Publish message event for ws-service to broadcast
	go s.publishMessageSentEvent(message, recipientIDs)

	// Step 8: Update unread counts for all recipients
	go func() {
		bgCtx := context.Background()
		for _, recipientID := range recipientIDs {
			if err := s.repo.UpdateParticipantUnreadCount(bgCtx, req.ConversationID, recipientID, true); err != nil {
				s.logger.Warn("Failed to update unread count",
					logger.String("user_id", recipientID.String()),
					logger.Error(err),
				)
			}
		}
	}()

	return message, nil
}

// publishMessageSentEvent publishes message.sent event for ws-service to handle
func (s *messageService) publishMessageSentEvent(message *models.Message, recipientIDs []uuid.UUID) {
	event := &domain.MessageEvent{
		EventType:      domain.MessageEventTypeSent,
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
		SenderUserID:   message.SenderUserID,
		RecipientIDs:   recipientIDs,
		Timestamp:      time.Now(),
		Payload:        message,
	}

	ctx := context.Background()
	if err := s.eventPublisher.PublishMessageEvent(ctx, event); err != nil {
		s.logger.Error("Failed to publish message.sent event",
			logger.String("message_id", message.ID.String()),
			logger.Error(err),
		)
	} else {
		s.logger.Debug("Published message.sent event",
			logger.String("message_id", message.ID.String()),
			logger.Int("recipients", len(recipientIDs)),
		)
	}

	// Also send push notifications for offline users (ws-service will handle online users)
	for _, recipientID := range recipientIDs {
		s.sendPushNotification(message, recipientID)
	}
}

// sendPushNotification sends a push notification for offline users via Kafka
func (s *messageService) sendPushNotification(message *models.Message, recipientID uuid.UUID) {
	notification := map[string]interface{}{
		"type":            "new_message",
		"user_id":         recipientID.String(),
		"message_id":      message.ID.String(),
		"conversation_id": message.ConversationID.String(),
		"sender_id":       message.SenderUserID.String(),
		"content":         message.Content,
		"message_type":    message.MessageType,
		"timestamp":       message.CreatedAt,
		"created_at":      time.Now(),
	}

	notifJSON, err := json.Marshal(notification)
	if err != nil {
		s.logger.Error("Failed to marshal notification",
			logger.String("message_id", message.ID.String()),
			logger.Error(err),
		)
		return
	}

	// Publish to Kafka topic for notification service
	kafkaMsg := messaging.NewMessage(notifJSON).
		WithKey([]byte(recipientID.String())).
		WithHeader("type", "notification").
		WithHeader("message_id", message.ID.String())

	if err := s.kafka.Send(context.Background(), "notifications", kafkaMsg); err != nil {
		s.logger.Error("Failed to publish notification",
			logger.String("message_id", message.ID.String()),
			logger.String("user_id", recipientID.String()),
			logger.Error(err),
		)
	} else {
		s.logger.Debug("Push notification sent",
			logger.String("message_id", message.ID.String()),
			logger.String("user_id", recipientID.String()),
		)
	}
}

// GetMessages retrieves messages for a conversation
func (s *messageService) GetMessages(ctx context.Context, conversationID uuid.UUID, params *models.PaginationParams) (*models.MessagesResponse, error) {
	messages, err := s.repo.GetMessages(ctx, conversationID, params)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get messages").
			WithService("message-service").
			WithDetail("conversation_id", conversationID.String())
	}

	hasMore := len(messages) == params.Limit

	return &models.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}

// GetMessage retrieves a single message
func (s *messageService) GetMessage(ctx context.Context, messageID uuid.UUID) (*models.Message, error) {
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}

// EditMessage edits an existing message
func (s *messageService) EditMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, newContent string) error {
	// Get original message to verify ownership
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	if message.SenderUserID != userID {
		return fmt.Errorf("unauthorized: only message sender can edit")
	}

	// Update message
	if err := s.repo.UpdateMessage(ctx, messageID, newContent); err != nil {
		s.logger.Error("Failed to edit message",
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		return fmt.Errorf("failed to edit message: %w", err)
	}

	// Publish edit event for ws-service to broadcast
	go func() {
		bgCtx := context.Background()
		participantIDs, err := s.repo.GetParticipantUserIDs(bgCtx, message.ConversationID)
		if err != nil {
			return
		}

		// Filter out the editor from recipients
		recipientIDs := make([]uuid.UUID, 0)
		for _, pid := range participantIDs {
			if pid != userID {
				recipientIDs = append(recipientIDs, pid)
			}
		}

		editEvent := &domain.MessageEvent{
			EventType:      domain.MessageEventTypeEdited,
			MessageID:      messageID,
			ConversationID: message.ConversationID,
			SenderUserID:   userID,
			RecipientIDs:   recipientIDs,
			Timestamp:      time.Now(),
			Payload: &models.Message{
				ID:      messageID,
				Content: newContent,
			},
		}

		if err := s.eventPublisher.PublishMessageEvent(bgCtx, editEvent); err != nil {
			s.logger.Error("Failed to publish message.edited event",
				logger.String("message_id", messageID.String()),
				logger.Error(err),
			)
		}
	}()

	return nil
}

// DeleteMessage deletes a message
func (s *messageService) DeleteMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error {
	if err := s.repo.DeleteMessage(ctx, messageID, userID); err != nil {
		s.logger.Error("Failed to delete message",
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		return fmt.Errorf("failed to delete message: %w", err)
	}

	// Publish delete event for ws-service to broadcast
	go func() {
		bgCtx := context.Background()
		message, err := s.repo.GetMessageByID(bgCtx, messageID)
		if err != nil {
			return
		}

		participantIDs, err := s.repo.GetParticipantUserIDs(bgCtx, message.ConversationID)
		if err != nil {
			return
		}

		// Filter out the deleter from recipients
		recipientIDs := make([]uuid.UUID, 0)
		for _, pid := range participantIDs {
			if pid != userID {
				recipientIDs = append(recipientIDs, pid)
			}
		}

		deleteEvent := &domain.MessageEvent{
			EventType:      domain.MessageEventTypeDeleted,
			MessageID:      messageID,
			ConversationID: message.ConversationID,
			SenderUserID:   userID,
			RecipientIDs:   recipientIDs,
			Timestamp:      time.Now(),
		}

		if err := s.eventPublisher.PublishMessageEvent(bgCtx, deleteEvent); err != nil {
			s.logger.Error("Failed to publish message.deleted event",
				logger.String("message_id", messageID.String()),
				logger.Error(err),
			)
		}
	}()

	return nil
}

// MarkAsDelivered marks a message as delivered
func (s *messageService) MarkAsDelivered(ctx context.Context, messageID, userID uuid.UUID) error {
	if err := s.repo.MarkAsDelivered(ctx, messageID, userID); err != nil {
		return fmt.Errorf("failed to mark as delivered: %w", err)
	}

	// Notify sender about delivery
	go s.notifyDeliveryStatus(messageID, userID, "delivered")

	return nil
}

// MarkAsRead marks a message as read
func (s *messageService) MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) error {
	if err := s.repo.MarkAsRead(ctx, messageID, userID); err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	// Notify sender about read receipt
	go s.notifyDeliveryStatus(messageID, userID, "read")

	return nil
}

// HandleReadReceipt processes read receipt from WebSocket
func (s *messageService) HandleReadReceipt(ctx context.Context, userID, messageID uuid.UUID) error {
	// Mark as read
	if err := s.MarkAsRead(ctx, messageID, userID); err != nil {
		s.logger.Error("Failed to mark message as read",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return err
	}

	// Get message to update conversation unread count
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err == nil {
		// Reset unread count for this conversation
		go func() {
			bgCtx := context.Background()
			if err := s.repo.ResetUnreadCount(bgCtx, message.ConversationID, userID); err != nil {
				s.logger.Warn("Failed to reset unread count",
					logger.String("conversation_id", message.ConversationID.String()),
					logger.String("user_id", userID.String()),
					logger.Error(err),
				)
			}
		}()
	}

	return nil
}

// notifyDeliveryStatus notifies sender about delivery/read status
func (s *messageService) notifyDeliveryStatus(messageID, readerID uuid.UUID, status string) {
	ctx := context.Background()

	// Get message to find sender
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return
	}

	// Don't notify if reader is the sender
	if message.SenderUserID == readerID {
		return
	}

	// Determine event type
	var eventType domain.MessageEventType
	if status == "delivered" {
		eventType = domain.MessageEventTypeDelivered
	} else if status == "read" {
		eventType = domain.MessageEventTypeRead
	} else {
		return
	}

	// Publish delivery/read event for ws-service
	event := &domain.MessageEvent{
		EventType:      eventType,
		MessageID:      messageID,
		ConversationID: message.ConversationID,
		SenderUserID:   readerID,
		RecipientIDs:   []uuid.UUID{message.SenderUserID},
		Timestamp:      time.Now(),
	}

	if err := s.eventPublisher.PublishMessageEvent(ctx, event); err != nil {
		s.logger.Debug("Failed to publish delivery status event",
			logger.String("message_id", messageID.String()),
			logger.String("status", status),
			logger.Error(err),
		)
	}
}

// SetTypingIndicator sets typing indicator for a user
func (s *messageService) SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) error {
	// Update in database
	if err := s.repo.SetTypingIndicator(ctx, conversationID, userID, isTyping); err != nil {
		return fmt.Errorf("failed to set typing indicator: %w", err)
	}

	// Publish typing event for ws-service to broadcast
	go func() {
		bgCtx := context.Background()
		participantIDs, err := s.repo.GetParticipantUserIDs(bgCtx, conversationID)
		if err != nil {
			return
		}

		// Filter out the typing user from recipients
		recipientIDs := make([]uuid.UUID, 0)
		for _, pid := range participantIDs {
			if pid != userID {
				recipientIDs = append(recipientIDs, pid)
			}
		}

		eventType := domain.ConversationEventTypeTypingStarted
		if !isTyping {
			eventType = domain.ConversationEventTypeTypingStopped
		}

		typingEvent := &domain.ConversationEvent{
			EventType:      eventType,
			ConversationID: conversationID,
			UserID:         userID,
			Timestamp:      time.Now(),
		}

		if err := s.eventPublisher.PublishConversationEvent(bgCtx, typingEvent); err != nil {
			s.logger.Error("Failed to publish typing event",
				logger.String("conversation_id", conversationID.String()),
				logger.Error(err),
			)
		}
	}()

	return nil
}

// MarkConversationAsRead marks all messages in a conversation as read
func (s *messageService) MarkConversationAsRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	// Reset unread count
	if err := s.repo.ResetUnreadCount(ctx, conversationID, userID); err != nil {
		return fmt.Errorf("failed to reset unread count: %w", err)
	}

	s.logger.Info("Conversation marked as read",
		logger.String("conversation_id", conversationID.String()),
		logger.String("user_id", userID.String()),
	)

	return nil
}
