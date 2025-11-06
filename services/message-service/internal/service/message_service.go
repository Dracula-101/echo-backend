package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"echo-backend/services/message-service/internal/model"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/websocket"

	"shared/pkg/circuitbreaker"
	"shared/pkg/logger"
	"shared/pkg/messaging"
	"shared/pkg/retry"

	"github.com/google/uuid"
)

type MessageService interface {
	SendMessage(ctx context.Context, req *model.SendMessageRequest) (*model.Message, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, params *model.PaginationParams) (*model.MessagesResponse, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*model.Message, error)
	EditMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, newContent string) error
	DeleteMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error
	MarkAsDelivered(ctx context.Context, messageID, userID uuid.UUID) error
	MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) error
	HandleReadReceipt(ctx context.Context, userID, messageID uuid.UUID) error
	SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) error
	MarkConversationAsRead(ctx context.Context, conversationID, userID uuid.UUID) error
}

type messageService struct {
	repo         repo.MessageRepository
	hub          *websocket.Hub
	kafka        messaging.Producer
	logger       logger.Logger
	dbCircuit    *circuitbreaker.CircuitBreaker
	kafkaCircuit *circuitbreaker.CircuitBreaker
	retryer      *retry.Retryer
}

func NewMessageService(
	repo repo.MessageRepository,
	hub *websocket.Hub,
	kafka messaging.Producer,
	log logger.Logger,
) MessageService {
	dbCircuit := circuitbreaker.New("message-db", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			log.Warn("Circuit breaker state changed",
				logger.String("name", name),
				logger.String("from", from.String()),
				logger.String("to", to.String()),
			)
		},
	})

	kafkaCircuit := circuitbreaker.New("message-kafka", circuitbreaker.Config{
		MaxRequests: 2,
		Interval:    5 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.InitialDelay = 100 * time.Millisecond
	retryConfig.MaxDelay = 2 * time.Second

	return &messageService{
		repo:         repo,
		hub:          hub,
		kafka:        kafka,
		logger:       log,
		dbCircuit:    dbCircuit,
		kafkaCircuit: kafkaCircuit,
		retryer:      retry.New(retryConfig),
	}
}

// SendMessage handles the complete flow of sending a message
// This is the core function - production-ready with all checks and broadcasts
func (s *messageService) SendMessage(ctx context.Context, req *model.SendMessageRequest) (*model.Message, error) {
	var canSend bool
	err := s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		var err error
		canSend, err = s.repo.ValidateParticipant(ctx, req.ConversationID, req.SenderUserID)
		return err
	})

	if err != nil {
		s.logger.Error("Failed to validate participant",
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("user_id", req.SenderUserID.String()),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to validate participant: %w", err)
	}

	if !canSend {
		s.logger.Warn("User not authorized to send message",
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("user_id", req.SenderUserID.String()),
		)
		return nil, fmt.Errorf("user not authorized to send messages in this conversation")
	}

	now := time.Now()
	message := &model.Message{
		ID:              uuid.New(),
		ConversationID:  req.ConversationID,
		SenderUserID:    req.SenderUserID,
		ParentMessageID: req.ParentMessageID,
		Content:         req.Content,
		MessageType:     req.MessageType,
		Status:          "sent",
		IsEdited:        false,
		IsDeleted:       false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if len(req.Mentions) > 0 {
		mentionsJSON, err := json.Marshal(req.Mentions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal mentions: %w", err)
		}
		message.Mentions = mentionsJSON
	}

	if req.Metadata != (model.Metadata{}) {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		message.Metadata = metadataJSON
	}

	err = s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
		return s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
			return s.repo.CreateMessage(ctx, message)
		})
	})

	if err != nil {
		s.logger.Error("Failed to create message",
			logger.String("message_id", message.ID.String()),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	s.logger.Info("Message created",
		logger.String("message_id", message.ID.String()),
		logger.String("conversation_id", message.ConversationID.String()),
		logger.String("sender_id", message.SenderUserID.String()),
	)

	var participantIDs []uuid.UUID
	err = s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		var err error
		participantIDs, err = s.repo.GetParticipantUserIDs(ctx, req.ConversationID)
		return err
	})

	if err != nil {
		s.logger.Error("Failed to get participants",
			logger.String("conversation_id", req.ConversationID.String()),
			logger.Error(err),
		)
		participantIDs = []uuid.UUID{}
	}

	go func() {
		bgCtx := context.Background()
		s.dbCircuit.ExecuteWithContext(bgCtx, func(ctx context.Context) error {
			return s.repo.UpdateConversationLastMessage(ctx, req.ConversationID, message.ID)
		})
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
			s.dbCircuit.ExecuteWithContext(bgCtx, func(ctx context.Context) error {
				return s.repo.CreateDeliveryStatus(ctx, message.ID, recipientIDs)
			})
		}()
	}

	// Step 7: Broadcast message to all participants
	go s.broadcastMessage(message, participantIDs, req.SenderUserID)

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

// broadcastMessage handles the intelligent broadcasting of messages
// Similar to WhatsApp: online users get WebSocket, offline get push notification
func (s *messageService) broadcastMessage(message *model.Message, participantIDs []uuid.UUID, senderID uuid.UUID) {
	event := model.MessageEvent{
		Type:      "new_message",
		Message:   message,
		Timestamp: time.Now(),
	}

	onlineCount := 0
	offlineCount := 0

	for _, participantID := range participantIDs {
		// Skip sender (they already have the message)
		if participantID == senderID {
			continue
		}

		// Check if user is online
		if s.hub.IsUserOnline(participantID) {
			// Send via WebSocket
			if err := s.hub.SendToUser(participantID, event); err != nil {
				s.logger.Error("Failed to send via WebSocket",
					logger.String("user_id", participantID.String()),
					logger.String("message_id", message.ID.String()),
					logger.Error(err),
				)
			} else {
				onlineCount++

				// Automatically mark as delivered for online users
				go func(uid uuid.UUID) {
					ctx := context.Background()
					if err := s.repo.MarkAsDelivered(ctx, message.ID, uid); err != nil {
						s.logger.Warn("Failed to mark as delivered",
							logger.String("message_id", message.ID.String()),
							logger.String("user_id", uid.String()),
							logger.Error(err),
						)
					}
				}(participantID)
			}
		} else {
			// User is offline, send push notification via Kafka
			s.sendPushNotification(message, participantID)
			offlineCount++
		}
	}

	s.logger.Info("Message broadcast complete",
		logger.String("message_id", message.ID.String()),
		logger.Int("online", onlineCount),
		logger.Int("offline", offlineCount),
		logger.Int("total", len(participantIDs)-1), // -1 for sender
	)
}

// sendPushNotification sends a push notification for offline users via Kafka
func (s *messageService) sendPushNotification(message *model.Message, recipientID uuid.UUID) {
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
func (s *messageService) GetMessages(ctx context.Context, conversationID uuid.UUID, params *model.PaginationParams) (*model.MessagesResponse, error) {
	messages, err := s.repo.GetMessages(ctx, conversationID, params)
	if err != nil {
		s.logger.Error("Failed to get messages",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	hasMore := len(messages) == params.Limit

	return &model.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}

// GetMessage retrieves a single message
func (s *messageService) GetMessage(ctx context.Context, messageID uuid.UUID) (*model.Message, error) {
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

	// Broadcast edit event to all participants
	go func() {
		bgCtx := context.Background()
		participantIDs, err := s.repo.GetParticipantUserIDs(bgCtx, message.ConversationID)
		if err != nil {
			return
		}

		editEvent := model.MessageEvent{
			Type:      "message_edited",
			MessageID: messageID,
			Message: &model.Message{
				ID:      messageID,
				Content: newContent,
			},
			Timestamp: time.Now(),
		}

		_ = s.hub.SendToUsers(participantIDs, editEvent, []uuid.UUID{userID})
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

	// Broadcast delete event
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

		deleteEvent := model.MessageEvent{
			Type:      "message_deleted",
			MessageID: messageID,
			UserID:    userID,
			Timestamp: time.Now(),
		}

		_ = s.hub.SendToUsers(participantIDs, deleteEvent, []uuid.UUID{userID})
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

	// Send notification to sender
	event := model.MessageEvent{
		Type:      fmt.Sprintf("message_%s", status),
		MessageID: messageID,
		UserID:    readerID,
		Timestamp: time.Now(),
	}

	if err := s.hub.SendToUser(message.SenderUserID, event); err != nil {
		s.logger.Debug("Failed to send delivery status to sender",
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

	// Broadcast to all participants except the typing user
	go func() {
		bgCtx := context.Background()
		participantIDs, err := s.repo.GetParticipantUserIDs(bgCtx, conversationID)
		if err != nil {
			return
		}

		typingEvent := model.MessageEvent{
			Type:      "typing",
			UserID:    userID,
			Timestamp: time.Now(),
		}

		_ = s.hub.SendToUsers(participantIDs, typingEvent, []uuid.UUID{userID})
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
