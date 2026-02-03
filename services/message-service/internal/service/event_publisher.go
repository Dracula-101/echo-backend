package service

import (
	"context"
	"encoding/json"

	"echo-backend/services/message-service/internal/domain"

	"shared/pkg/logger"
	"shared/pkg/messaging"
)

// EventPublisher publishes domain events to message broker
type EventPublisher interface {
	PublishMessageEvent(ctx context.Context, event *domain.MessageEvent) error
	PublishChatMessage(ctx context.Context, event *domain.ChatMessageEvent) error
	PublishConversationEvent(ctx context.Context, event *domain.ConversationEvent) error
	PublishDeliveryEvent(ctx context.Context, event *DeliveryEvent) error
}

// KafkaEventPublisher implements EventPublisher using Kafka
type KafkaEventPublisher struct {
	producer messaging.Producer
	log      logger.Logger
}

func NewKafkaEventPublisher(producer messaging.Producer, log logger.Logger) *KafkaEventPublisher {
	return &KafkaEventPublisher{
		producer: producer,
		log:      log,
	}
}

func (p *KafkaEventPublisher) PublishMessageEvent(ctx context.Context, event *domain.MessageEvent) error {
	topic := "message-events"
	key := event.ConversationID.String()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		p.log.Error("Failed to marshal message event",
			logger.Error(err),
			logger.String("event_type", string(event.EventType)),
			logger.String("message_id", event.MessageID.String()),
		)
		return err
	}

	kafkaMsg := messaging.NewMessage(eventJSON).
		WithKey([]byte(key)).
		WithHeader("type", "message_event").
		WithHeader("event_type", string(event.EventType)).
		WithHeader("message_id", event.MessageID.String())

	err = p.producer.Send(ctx, topic, kafkaMsg)
	if err != nil {
		p.log.Error("Failed to publish message event",
			logger.Error(err),
			logger.String("event_type", string(event.EventType)),
			logger.String("message_id", event.MessageID.String()),
		)
		return err
	}

	p.log.Debug("Published message event",
		logger.String("event_type", string(event.EventType)),
		logger.String("message_id", event.MessageID.String()),
	)

	return nil
}

func (p *KafkaEventPublisher) PublishChatMessage(ctx context.Context, event *domain.ChatMessageEvent) error {
	topic := "chat-messages"
	key := event.ConversationID.String()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		p.log.Error("Failed to marshal chat message event",
			logger.Error(err),
			logger.String("conversation_id", event.ConversationID.String()),
		)
		return err
	}

	kafkaMsg := messaging.NewMessage(eventJSON).
		WithKey([]byte(key)).
		WithHeader("type", "chat_message").
		WithHeader("conversation_id", event.ConversationID.String())

	err = p.producer.Send(ctx, topic, kafkaMsg)
	if err != nil {
		p.log.Error("Failed to publish chat message event",
			logger.Error(err),
			logger.String("conversation_id", event.ConversationID.String()),
		)
		return err
	}

	p.log.Debug("Published chat message event",
		logger.String("conversation_id", event.ConversationID.String()),
	)

	return nil
}

func (p *KafkaEventPublisher) PublishConversationEvent(ctx context.Context, event *domain.ConversationEvent) error {
	topic := "conversation-events"
	key := event.ConversationID.String()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		p.log.Error("Failed to marshal conversation event",
			logger.Error(err),
			logger.String("event_type", string(event.EventType)),
		)
		return err
	}

	kafkaMsg := messaging.NewMessage(eventJSON).
		WithKey([]byte(key)).
		WithHeader("type", "conversation_event").
		WithHeader("event_type", string(event.EventType)).
		WithHeader("conversation_id", event.ConversationID.String())

	err = p.producer.Send(ctx, topic, kafkaMsg)
	if err != nil {
		p.log.Error("Failed to publish conversation event",
			logger.Error(err),
			logger.String("event_type", string(event.EventType)),
		)
		return err
	}

	p.log.Debug("Published conversation event",
		logger.String("event_type", string(event.EventType)),
		logger.String("conversation_id", event.ConversationID.String()),
	)

	return nil
}

func (p *KafkaEventPublisher) PublishDeliveryEvent(ctx context.Context, event *DeliveryEvent) error {
	topic := "delivery-events"
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

	err = p.producer.Send(ctx, topic, kafkaMsg)
	if err != nil {
		p.log.Error("Failed to publish delivery event",
			logger.Error(err),
			logger.String("event_type", string(event.EventType)),
			logger.String("user_id", event.UserID.String()),
		)
		return err
	}

	p.log.Debug("Published delivery event",
		logger.String("event_type", string(event.EventType)),
		logger.String("user_id", event.UserID.String()),
		logger.Int("message_count", len(event.MessageIDs)),
	)

	return nil
}
