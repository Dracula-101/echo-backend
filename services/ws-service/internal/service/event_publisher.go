package service

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/pkg/messaging"

	"github.com/google/uuid"
)

// DeliveryEventType represents the type of delivery event
type DeliveryEventType string

const (
	DeliveryEventDelivered DeliveryEventType = "delivered"
	DeliveryEventRead      DeliveryEventType = "read"
)

// DeliveryEvent represents a delivery status update event to be published to Kafka
type DeliveryEvent struct {
	EventType      DeliveryEventType `json:"event_type"`
	MessageIDs     []uuid.UUID       `json:"message_ids"`
	UserID         uuid.UUID         `json:"user_id"`
	ConversationID uuid.UUID         `json:"conversation_id"`
	Timestamp      time.Time         `json:"timestamp"`
}

// DeliveryEventPublisher publishes delivery events to Kafka
type DeliveryEventPublisher interface {
	PublishDeliveryEvent(ctx context.Context, event *DeliveryEvent) error
}

// KafkaDeliveryPublisher implements DeliveryEventPublisher using Kafka
type KafkaDeliveryPublisher struct {
	producer messaging.Producer
	topic    string
	log      logger.Logger
}

// NewKafkaDeliveryPublisher creates a new Kafka delivery event publisher
func NewKafkaDeliveryPublisher(producer messaging.Producer, topic string, log logger.Logger) *KafkaDeliveryPublisher {
	if topic == "" {
		topic = "delivery-events"
	}

	return &KafkaDeliveryPublisher{
		producer: producer,
		topic:    topic,
		log:      log,
	}
}

// PublishDeliveryEvent publishes a delivery event to Kafka
func (p *KafkaDeliveryPublisher) PublishDeliveryEvent(ctx context.Context, event *DeliveryEvent) error {
	key := event.ConversationID.String()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		p.log.Error("Failed to marshal delivery event",
			logger.Error(err),
			logger.String("event_type", string(event.EventType)),
			logger.String("user_id", event.UserID.String()),
		)
		return err
	}

	kafkaMsg := messaging.NewMessage(eventJSON).
		WithKey([]byte(key)).
		WithHeader("type", "delivery_event").
		WithHeader("event_type", string(event.EventType)).
		WithHeader("user_id", event.UserID.String())

	err = p.producer.Send(ctx, p.topic, kafkaMsg)
	if err != nil {
		p.log.Error("Failed to publish delivery event",
			logger.Error(err),
			logger.String("topic", p.topic),
			logger.String("event_type", string(event.EventType)),
			logger.String("user_id", event.UserID.String()),
		)
		return err
	}

	p.log.Debug("Published delivery event",
		logger.String("topic", p.topic),
		logger.String("event_type", string(event.EventType)),
		logger.String("user_id", event.UserID.String()),
		logger.Int("message_count", len(event.MessageIDs)),
	)

	return nil
}
