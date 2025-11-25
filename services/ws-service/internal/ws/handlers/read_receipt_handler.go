package handlers

import (
	"context"
	"encoding/json"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/broadcast"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

// MarkAsReadHandler handles mark as read messages
type MarkAsReadHandler struct {
	broadcaster *broadcast.Broadcaster
	log         logger.Logger
}

// NewMarkAsReadHandler creates a new mark as read handler
func NewMarkAsReadHandler(broadcaster *broadcast.Broadcaster, log logger.Logger) *MarkAsReadHandler {
	return &MarkAsReadHandler{
		broadcaster: broadcaster,
		log:         log,
	}
}

// Handle handles a mark as read message
func (h *MarkAsReadHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.ReadReceiptPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Debug("Processing mark as read",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("conversation_id", payload.ConversationID.String()),
		logger.Int("message_count", len(payload.MessageIDs)),
	)

	// TODO: Update message read status in database
	// Broadcast read receipt to conversation participants
	broadcastMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeMessageRead,
		Payload: map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": payload.ConversationID,
			"message_ids":     payload.MessageIDs,
			"read_at":         time.Now(),
		},
		Timestamp: time.Now(),
	}

	h.broadcaster.BroadcastToTopic(protocol.TopicConversation, payload.ConversationID.String(), broadcastMsg, client.UserID)

	return nil
}

// MessageType returns the message type this handler handles
func (h *MarkAsReadHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeMarkAsRead
}

// MarkAsDeliveredHandler handles mark as delivered messages
type MarkAsDeliveredHandler struct{
	broadcaster *broadcast.Broadcaster
	log         logger.Logger
}

// NewMarkAsDeliveredHandler creates a new mark as delivered handler
func NewMarkAsDeliveredHandler(broadcaster *broadcast.Broadcaster, log logger.Logger) *MarkAsDeliveredHandler {
	return &MarkAsDeliveredHandler{
		broadcaster: broadcaster,
		log:         log,
	}
}

// Handle handles a mark as delivered message
func (h *MarkAsDeliveredHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.ReadReceiptPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Debug("Processing mark as delivered",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("conversation_id", payload.ConversationID.String()),
		logger.Int("message_count", len(payload.MessageIDs)),
	)

	// TODO: Update message delivery status in database
	// Broadcast delivery receipt
	broadcastMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeMessageDelivered,
		Payload: map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": payload.ConversationID,
			"message_ids":     payload.MessageIDs,
			"delivered_at":    time.Now(),
		},
		Timestamp: time.Now(),
	}

	h.broadcaster.BroadcastToTopic(protocol.TopicConversation, payload.ConversationID.String(), broadcastMsg, client.UserID)

	return nil
}

// MessageType returns the message type this handler handles
func (h *MarkAsDeliveredHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeMarkAsDelivered
}
