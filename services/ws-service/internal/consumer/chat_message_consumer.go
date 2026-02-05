package consumer

import (
	"context"
	"encoding/json"

	"shared/pkg/logger"
	"shared/pkg/messaging"
	"ws-service/internal/domain"
	"ws-service/internal/protocol"
	"ws-service/websocket"

	"github.com/google/uuid"
)

// ChatMessageConsumer consumes chat messages from Kafka and broadcasts to WebSocket clients
type ChatMessageConsumer struct {
	manager *websocket.Manager
	logger  logger.Logger
}

// NewChatMessageConsumer creates a new chat message consumer
func NewChatMessageConsumer(manager *websocket.Manager, log logger.Logger) *ChatMessageConsumer {
	return &ChatMessageConsumer{
		manager: manager,
		logger:  log,
	}
}

// Handle processes a single chat message from Kafka
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

	c.logger.Debug("processing chat message for WebSocket delivery",
		logger.String("message_id", event.ID.String()),
		logger.String("conversation_id", event.ConversationID.String()),
		logger.String("sender_id", event.SenderUserID.String()),
	)

	payload := event.ToPayload()

	// Send message.sending to the sender to confirm the message is being processed
	if err := c.manager.SendEventToUser(event.SenderUserID, protocol.MsgTypeMessageSending, &MessageSendingPayload{
		MessageID:      event.ID,
		ConversationID: event.ConversationID,
		Status:         "sending",
	}); err != nil {
		c.logger.Warn("failed to send message.sending to sender",
			logger.String("message_id", event.ID.String()),
			logger.String("sender_id", event.SenderUserID.String()),
			logger.Error(err),
		)
		// Don't fail - sender might be offline
	}

	// Broadcast message.new to all conversation participants (excluding sender)
	if broadcastErr := c.manager.BroadcastToConversation(
		event.ConversationID,
		string(protocol.MsgTypeMessageNew),
		payload,
		event.SenderUserID,
	); broadcastErr != nil {
		c.logger.Error("failed to broadcast message to conversation",
			logger.String("message_id", event.ID.String()),
			logger.String("conversation_id", event.ConversationID.String()),
			logger.Error(broadcastErr),
		)
		return broadcastErr
	} else {
		// send message.sent to the sender to confirm delivery
		if err := c.manager.SendEventToUser(event.SenderUserID, protocol.MsgTypeMessageDelivered, payload); err != nil {
			c.logger.Warn("failed to send message.sent to sender",
				logger.String("message_id", event.ID.String()),
				logger.String("sender_id", event.SenderUserID.String()),
				logger.Error(err),
			)
		}
	}

	c.logger.Debug("successfully broadcast message to conversation",
		logger.String("message_id", event.ID.String()),
		logger.String("conversation_id", event.ConversationID.String()),
	)

	return nil
}

// MessageSendingPayload is sent to the sender to confirm message is being processed
type MessageSendingPayload struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Status         string    `json:"status"`
}
