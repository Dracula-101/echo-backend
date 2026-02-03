package service

import (
	"context"
	"time"

	"echo-backend/services/message-service/internal/repo"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// DeliveryServiceInterface defines the contract for delivery tracking operations
type DeliveryServiceInterface interface {
	// CreateDeliveryStatuses creates initial delivery status records for message recipients
	CreateDeliveryStatuses(ctx context.Context, messageID uuid.UUID, recipientIDs []uuid.UUID) error

	// MarkAsDelivered marks a message as delivered to a specific user
	MarkAsDelivered(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error

	// MarkAsDeliveredBatch marks multiple messages as delivered to a specific user
	MarkAsDeliveredBatch(ctx context.Context, messageIDs []uuid.UUID, userID uuid.UUID) error

	// MarkAsRead marks a message as read by a specific user
	MarkAsRead(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error

	// MarkAsReadBatch marks multiple messages as read by a specific user
	MarkAsReadBatch(ctx context.Context, messageIDs []uuid.UUID, userID uuid.UUID) error

	// ProcessDeliveryEvent processes a delivery event from Kafka
	ProcessDeliveryEvent(ctx context.Context, event *DeliveryEvent) error
}

// DeliveryEvent represents a delivery status update event from Kafka
type DeliveryEvent struct {
	EventType      DeliveryEventType `json:"event_type"`
	MessageIDs     []uuid.UUID       `json:"message_ids"`
	UserID         uuid.UUID         `json:"user_id"`
	ConversationID uuid.UUID         `json:"conversation_id"`
	Timestamp      time.Time         `json:"timestamp"`
}

// DeliveryEventType represents the type of delivery event
type DeliveryEventType string

const (
	DeliveryEventDelivered DeliveryEventType = "delivered"
	DeliveryEventRead      DeliveryEventType = "read"
)

type deliveryService struct {
	messageRepo repo.MessageRepository
	logger      logger.Logger
}

// NewDeliveryService creates a new delivery service instance
func NewDeliveryService(messageRepo repo.MessageRepository, log logger.Logger) DeliveryServiceInterface {
	return &deliveryService{
		messageRepo: messageRepo,
		logger:      log,
	}
}

// CreateDeliveryStatuses creates initial delivery status records for message recipients
func (s *deliveryService) CreateDeliveryStatuses(ctx context.Context, messageID uuid.UUID, recipientIDs []uuid.UUID) error {
	if len(recipientIDs) == 0 {
		return nil
	}

	recipientStrings := make([]string, len(recipientIDs))
	for i, id := range recipientIDs {
		recipientStrings[i] = id.String()
	}

	if err := s.messageRepo.CreateDeliveryStatuses(ctx, messageID.String(), recipientStrings); err != nil {
		s.logger.Error("Failed to create delivery statuses",
			logger.String("message_id", messageID.String()),
			logger.Int("recipient_count", len(recipientIDs)),
			logger.Error(err),
		)
		return err
	}

	s.logger.Debug("Delivery statuses created",
		logger.String("message_id", messageID.String()),
		logger.Int("recipient_count", len(recipientIDs)),
	)

	return nil
}

// MarkAsDelivered marks a message as delivered to a specific user
func (s *deliveryService) MarkAsDelivered(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()

	// Update delivery status record
	if err := s.messageRepo.UpdateDeliveryStatus(ctx, messageID.String(), userID.String(), "delivered", &now); err != nil {
		s.logger.Error("Failed to update delivery status",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return err
	}

	// Increment delivery count on the message
	if err := s.messageRepo.IncrementMessageDeliveryCount(ctx, messageID.String()); err != nil {
		s.logger.Error("Failed to increment delivery count",
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		// Don't return error - the status update succeeded
	}

	s.logger.Debug("Message marked as delivered",
		logger.String("message_id", messageID.String()),
		logger.String("user_id", userID.String()),
	)

	return nil
}

// MarkAsDeliveredBatch marks multiple messages as delivered to a specific user
func (s *deliveryService) MarkAsDeliveredBatch(ctx context.Context, messageIDs []uuid.UUID, userID uuid.UUID) error {
	if len(messageIDs) == 0 {
		return nil
	}

	now := time.Now()
	messageIDStrings := make([]string, len(messageIDs))
	for i, id := range messageIDs {
		messageIDStrings[i] = id.String()
	}

	// Update delivery status records
	if err := s.messageRepo.UpdateDeliveryStatusBatch(ctx, messageIDStrings, userID.String(), "delivered", &now); err != nil {
		s.logger.Error("Failed to update delivery status batch",
			logger.Int("message_count", len(messageIDs)),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return err
	}

	// Increment delivery count for each message
	for _, msgID := range messageIDs {
		if err := s.messageRepo.IncrementMessageDeliveryCount(ctx, msgID.String()); err != nil {
			s.logger.Error("Failed to increment delivery count",
				logger.String("message_id", msgID.String()),
				logger.Error(err),
			)
			// Continue with other messages
		}
	}

	s.logger.Debug("Messages marked as delivered",
		logger.Int("message_count", len(messageIDs)),
		logger.String("user_id", userID.String()),
	)

	return nil
}

// MarkAsRead marks a message as read by a specific user
func (s *deliveryService) MarkAsRead(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()

	// Update delivery status record
	if err := s.messageRepo.UpdateDeliveryStatus(ctx, messageID.String(), userID.String(), "read", &now); err != nil {
		s.logger.Error("Failed to update read status",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return err
	}

	// Increment read count on the message
	if err := s.messageRepo.IncrementMessageReadCount(ctx, messageID.String()); err != nil {
		s.logger.Error("Failed to increment read count",
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		// Don't return error - the status update succeeded
	}

	s.logger.Debug("Message marked as read",
		logger.String("message_id", messageID.String()),
		logger.String("user_id", userID.String()),
	)

	return nil
}

// MarkAsReadBatch marks multiple messages as read by a specific user
func (s *deliveryService) MarkAsReadBatch(ctx context.Context, messageIDs []uuid.UUID, userID uuid.UUID) error {
	if len(messageIDs) == 0 {
		return nil
	}

	now := time.Now()
	messageIDStrings := make([]string, len(messageIDs))
	for i, id := range messageIDs {
		messageIDStrings[i] = id.String()
	}

	// Update delivery status records
	if err := s.messageRepo.UpdateDeliveryStatusBatch(ctx, messageIDStrings, userID.String(), "read", &now); err != nil {
		s.logger.Error("Failed to update read status batch",
			logger.Int("message_count", len(messageIDs)),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return err
	}

	// Increment read count for each message
	for _, msgID := range messageIDs {
		if err := s.messageRepo.IncrementMessageReadCount(ctx, msgID.String()); err != nil {
			s.logger.Error("Failed to increment read count",
				logger.String("message_id", msgID.String()),
				logger.Error(err),
			)
			// Continue with other messages
		}
	}

	s.logger.Debug("Messages marked as read",
		logger.Int("message_count", len(messageIDs)),
		logger.String("user_id", userID.String()),
	)

	return nil
}

// ProcessDeliveryEvent processes a delivery event from Kafka
func (s *deliveryService) ProcessDeliveryEvent(ctx context.Context, event *DeliveryEvent) error {
	if event == nil || len(event.MessageIDs) == 0 {
		return pkgErrors.New(pkgErrors.CodeInvalidArgument, "invalid delivery event")
	}

	switch event.EventType {
	case DeliveryEventDelivered:
		return s.MarkAsDeliveredBatch(ctx, event.MessageIDs, event.UserID)
	case DeliveryEventRead:
		return s.MarkAsReadBatch(ctx, event.MessageIDs, event.UserID)
	default:
		s.logger.Warn("Unknown delivery event type",
			logger.String("event_type", string(event.EventType)),
		)
		return pkgErrors.New(pkgErrors.CodeInvalidArgument, "unknown delivery event type")
	}
}
