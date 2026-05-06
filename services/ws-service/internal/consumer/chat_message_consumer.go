package consumer

import (
	"context"
	"encoding/json"

	"shared/pkg/logger"
	"shared/pkg/messaging"
	"ws-service/internal/dedup"
	"ws-service/internal/dlq"
	"ws-service/internal/domain"
	"ws-service/internal/protocol"
	"ws-service/internal/repo"
	"ws-service/websocket"

	"github.com/google/uuid"
)

// ChatMessageConsumer consumes chat messages from Kafka and broadcasts to WebSocket clients
type ChatMessageConsumer struct {
	manager      *websocket.Manager
	logger       logger.Logger
	dedup        *dedup.Tracker
	deliveryRepo repo.DeliveryStatusRepository
	dlq          *dlq.Publisher
}

// NewChatMessageConsumer creates a new chat message consumer
func NewChatMessageConsumer(manager *websocket.Manager, deduper *dedup.Tracker, deliveryRepo repo.DeliveryStatusRepository, dlqPub *dlq.Publisher, log logger.Logger) *ChatMessageConsumer {
	return &ChatMessageConsumer{
		manager:      manager,
		logger:       log,
		dedup:        deduper,
		deliveryRepo: deliveryRepo,
		dlq:          dlqPub,
	}
}

// Handle processes a single chat message from Kafka
func (c *ChatMessageConsumer) Handle(ctx context.Context, message *messaging.Message) error {
	var event domain.ChatMessageEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		c.logger.Error("failed to unmarshal chat message event — sending to DLQ",
			logger.Error(err),
			logger.String("topic", message.Topic),
			logger.Int64("offset", message.Offset),
		)
		c.dlq.Send(ctx, message, "unmarshal_failed: "+err.Error())
		// Returning nil lets the consumer mark+commit; the poisoned record is
		// preserved on the DLQ for inspection.
		return nil
	}

	if c.dedup != nil && !c.dedup.SeenWithin(event.ID.String()) {
		c.logger.Debug("skipping duplicate chat message",
			logger.String("message_id", event.ID.String()),
		)
		return nil
	}

	c.logger.Debug("processing chat message for WebSocket delivery",
		logger.String("message_id", event.ID.String()),
		logger.String("conversation_id", event.ConversationID.String()),
		logger.String("sender_id", event.SenderUserID.String()),
	)

	if c.deliveryRepo != nil && len(event.RecipientIDs) > 0 {
		// Belt-and-braces against the messages.create_delivery_status trigger:
		// ON CONFLICT DO NOTHING means a duplicate insert is harmless, but we
		// recover automatically if the trigger ever drifts or is disabled.
		if err := c.deliveryRepo.EnsureDeliveryStatus(ctx, event.ID, event.RecipientIDs); err != nil {
			c.logger.Warn("delivery_status safety-net insert failed",
				logger.String("message_id", event.ID.String()),
				logger.Error(err),
			)
		}
	}

	payload := event.ToPayload()

	// Stamp a per-conversation sequence number so reconnecting clients can ask
	// for "everything after seq N" without time-window guesswork.
	if seqCounter := c.manager.SeqCounter(); seqCounter != nil {
		if seq, err := seqCounter.Next(ctx, event.ConversationID.String()); err == nil {
			payload.Seq = seq
		}
	}

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
