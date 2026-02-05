package consumer

import (
	"context"
	"encoding/json"

	"echo-backend/services/message-service/internal/domain"
	"echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
	"shared/pkg/messaging"
)

// ChatMessageConsumer consumes incoming chat message events from Kafka
type ChatMessageConsumer struct {
	messageService service.MessageServiceInterface
	logger         logger.Logger
}

// NewChatMessageConsumer creates a new ChatMessageConsumer
func NewChatMessageConsumer(
	messageService service.MessageServiceInterface,
	log logger.Logger,
) *ChatMessageConsumer {
	return &ChatMessageConsumer{
		messageService: messageService,
		logger:         log,
	}
}

func (c *ChatMessageConsumer) Handle(ctx context.Context, message *messaging.Message) error {
	var event domain.ChatMessageEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		c.logger.Error("failed to unmarshal chat message event",
			logger.Error(err),
			logger.String("topic", message.Topic),
			logger.Int64("offset", message.Offset),
		)
		return err
	}

	c.logger.Debug("processing chat message event",
		logger.String("message_id", event.ID.String()),
		logger.String("conversation_id", event.ConversationID.String()),
	)

	if err := c.messageService.ProcessChatMessage(ctx, &event); err != nil {
		c.logger.Error("failed to process chat message",
			logger.String("message_id", event.ID.String()),
			logger.String("conversation_id", event.ConversationID.String()),
			logger.Error(err),
		)
		return err
	}

	c.logger.Debug("chat message event processed successfully",
		logger.String("message_id", event.ID.String()),
		logger.String("conversation_id", event.ConversationID.String()),
	)

	return nil
}

func (c *ChatMessageConsumer) HandleBatch(ctx context.Context, messages []*messaging.Message) error {
	var events []*domain.ChatMessageEvent
	var skipped int
	for _, message := range messages {
		var event domain.ChatMessageEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			c.logger.Error("failed to unmarshal chat message event in batch, skipping",
				logger.Error(err),
				logger.String("topic", message.Topic),
				logger.Int64("offset", message.Offset),
			)
			skipped++
			continue
		}
		events = append(events, &event)
	}

	if len(events) == 0 {
		c.logger.Warn("no valid events in batch",
			logger.Int("total", len(messages)),
			logger.Int("skipped", skipped),
		)
		return nil
	}

	c.logger.Debug("processing batch of chat message events",
		logger.Int("batch_size", len(events)),
		logger.Int("skipped", skipped),
	)

	if err := c.messageService.ProcessChatMessages(ctx, events); err != nil {
		c.logger.Error("failed to process batch of chat messages",
			logger.Int("batch_size", len(events)),
			logger.Error(err),
		)
		return err
	}

	c.logger.Debug("batch of chat message events processed successfully",
		logger.Int("batch_size", len(events)),
	)

	return nil
}
