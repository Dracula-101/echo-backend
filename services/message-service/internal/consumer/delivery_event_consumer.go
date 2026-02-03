package consumer

import (
	"context"
	"encoding/json"

	"echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
	"shared/pkg/messaging"
)

// DeliveryEventConsumer consumes delivery events from Kafka
type DeliveryEventConsumer struct {
	deliveryService service.DeliveryServiceInterface
	logger          logger.Logger
}

// NewDeliveryEventConsumer creates a new delivery event consumer
func NewDeliveryEventConsumer(
	deliveryService service.DeliveryServiceInterface,
	log logger.Logger,
) *DeliveryEventConsumer {
	return &DeliveryEventConsumer{
		deliveryService: deliveryService,
		logger:          log,
	}
}

// Handle processes a single delivery event from Kafka
func (c *DeliveryEventConsumer) Handle(ctx context.Context, message *messaging.Message) error {
	var event service.DeliveryEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		c.logger.Error("failed to unmarshal delivery event",
			logger.Error(err),
			logger.String("topic", message.Topic),
			logger.Int64("offset", message.Offset),
		)
		return err
	}

	c.logger.Debug("processing delivery event",
		logger.String("event_type", string(event.EventType)),
		logger.String("user_id", event.UserID.String()),
		logger.Int("message_count", len(event.MessageIDs)),
	)

	if err := c.deliveryService.ProcessDeliveryEvent(ctx, &event); err != nil {
		c.logger.Error("failed to process delivery event",
			logger.String("event_type", string(event.EventType)),
			logger.String("user_id", event.UserID.String()),
			logger.Error(err),
		)
		return err
	}

	c.logger.Debug("delivery event processed successfully",
		logger.String("event_type", string(event.EventType)),
		logger.String("user_id", event.UserID.String()),
	)

	return nil
}
