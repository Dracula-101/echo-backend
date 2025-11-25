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

// TypingStartHandler handles typing start messages
type TypingStartHandler struct {
	broadcaster *broadcast.Broadcaster
	hub         *websocket.Hub
	log         logger.Logger
}

// NewTypingStartHandler creates a new typing start handler
func NewTypingStartHandler(broadcaster *broadcast.Broadcaster, hub *websocket.Hub, log logger.Logger) *TypingStartHandler {
	return &TypingStartHandler{
		broadcaster: broadcaster,
		hub:         hub,
		log:         log,
	}
}

// Handle handles a typing start message
func (h *TypingStartHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.TypingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Debug("Processing typing start",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("conversation_id", payload.ConversationID.String()),
	)

	// TODO: Get conversation participants from database or cache
	// For now, broadcast to topic subscribers
	broadcastMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeTypingStart,
		Payload: map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": payload.ConversationID,
		},
		Timestamp: time.Now(),
	}

	// Broadcast to conversation topic
	h.broadcaster.BroadcastToTopic(protocol.TopicConversation, payload.ConversationID.String(), broadcastMsg, client.UserID)

	return nil
}

// MessageType returns the message type this handler handles
func (h *TypingStartHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeTypingStart
}

// TypingStopHandler handles typing stop messages
type TypingStopHandler struct {
	broadcaster *broadcast.Broadcaster
	hub         *websocket.Hub
	log         logger.Logger
}

// NewTypingStopHandler creates a new typing stop handler
func NewTypingStopHandler(broadcaster *broadcast.Broadcaster, hub *websocket.Hub, log logger.Logger) *TypingStopHandler {
	return &TypingStopHandler{
		broadcaster: broadcaster,
		hub:         hub,
		log:         log,
	}
}

// Handle handles a typing stop message
func (h *TypingStopHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.TypingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Debug("Processing typing stop",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("conversation_id", payload.ConversationID.String()),
	)

	broadcastMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeTypingStop,
		Payload: map[string]interface{}{
			"user_id":         client.UserID,
			"conversation_id": payload.ConversationID,
		},
		Timestamp: time.Now(),
	}

	h.broadcaster.BroadcastToTopic(protocol.TopicConversation, payload.ConversationID.String(), broadcastMsg, client.UserID)

	return nil
}

// MessageType returns the message type this handler handles
func (h *TypingStopHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeTypingStop
}

func generateBroadcastID() string {
	return "bcast_" + time.Now().Format("20060102150405")
}
