package handlers

import (
	"context"
	"encoding/json"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/protocol"
	"ws-service/internal/ws/subscription"

	"shared/pkg/logger"
)

// UnsubscribeHandler handles unsubscribe messages
type UnsubscribeHandler struct {
	subManager *subscription.Manager
	log        logger.Logger
}

// NewUnsubscribeHandler creates a new unsubscribe handler
func NewUnsubscribeHandler(subManager *subscription.Manager, log logger.Logger) *UnsubscribeHandler {
	return &UnsubscribeHandler{
		subManager: subManager,
		log:        log,
	}
}

// Handle handles an unsubscribe message
func (h *UnsubscribeHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.UnsubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Info("Processing unsubscription",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.Int("topic_count", len(payload.Topics)),
	)

	// Unsubscribe from topics
	for _, topic := range payload.Topics {
		h.subManager.Unsubscribe(client, topic)
	}

	// Send success response
	successMsg := protocol.ServerMessage{
		ID:        generateResponseID(msg.ID),
		Type:      protocol.ServerTypeUnsubscribed,
		RequestID: msg.ID,
		Payload: map[string]interface{}{
			"topics": payload.Topics,
		},
		Timestamp: time.Now(),
	}

	return client.SendMessage(successMsg)
}

// MessageType returns the message type this handler handles
func (h *UnsubscribeHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeUnsubscribe
}
