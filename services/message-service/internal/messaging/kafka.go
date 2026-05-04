package messaging

import (
	"context"
	"encoding/json"

	"echo-backend/services/message-service/internal/domain"
	"shared/pkg/logger"
	"shared/pkg/messaging"
)

type KafkaEventPublisher struct {
	producer      messaging.Producer
	deliveryTopic string
	log           logger.Logger
}

func NewKafkaEventPublisher(producer messaging.Producer, deliveryTopic string, log logger.Logger) *KafkaEventPublisher {
	if deliveryTopic == "" {
		deliveryTopic = "delivery-events"
	}

	return &KafkaEventPublisher{
		producer:      producer,
		deliveryTopic: deliveryTopic,
		log:           log,
	}
}

func (p *KafkaEventPublisher) publish(
	ctx context.Context,
	topic string,
	key string,
	eventType string,
	headers map[string]string,
	payload any,
) error {
	body, err := json.Marshal(payload)
	if err != nil {
		p.log.Error("kafka-publisher: failed to marshal payload",
			logger.Error(err),
			logger.String("topic", topic),
			logger.String("event_type", eventType),
		)
		return err
	}

	msg := messaging.NewMessage(body).
		WithKey([]byte(key)).
		WithHeader("event_type", eventType).
		WithHeaders(headers)

	if err = p.producer.Send(ctx, topic, msg); err != nil {
		p.log.Error("kafka-publisher: failed to send",
			logger.Error(err),
			logger.String("topic", topic),
			logger.String("event_type", eventType),
		)
		return err
	}

	p.log.Debug("kafka-publisher: event published",
		logger.String("topic", topic),
		logger.String("event_type", eventType),
	)
	return nil
}

func (p *KafkaEventPublisher) PublishChatMessage(ctx context.Context, event *domain.ChatMessageEvent) error {
	return p.publish(ctx, "chat-messages", event.ConversationID.String(), "chat_message",
		map[string]string{
			"type":            "chat_message",
			"conversation_id": event.ConversationID.String(),
		},
		event,
	)
}

func (p *KafkaEventPublisher) PublishMessageEvent(ctx context.Context, event *domain.MessageEvent) error {
	return p.publish(ctx, "message-events", event.MessageID.String(), string(event.EventType),
		map[string]string{
			"type":            "message_event",
			"conversation_id": event.ConversationID.String(),
			"user_id":         event.SenderUserID.String(),
		},
		event,
	)
}

func (p *KafkaEventPublisher) PublishDeliveryEvent(ctx context.Context, event *DeliveryEvent) error {
	return p.publish(ctx, p.deliveryTopic, event.ConversationID.String(), string(event.EventType),
		map[string]string{
			"type":    "delivery_event",
			"user_id": event.UserID.String(),
		},
		event,
	)
}

func (p *KafkaEventPublisher) PublishConversationEvent(ctx context.Context, event *domain.ConversationEvent) error {
	return p.publish(ctx, "conversation-events", event.ConversationID.String(), string(event.EventType),
		map[string]string{
			"type":            "conversation_event",
			"conversation_id": event.ConversationID.String(),
		},
		event,
	)
}
