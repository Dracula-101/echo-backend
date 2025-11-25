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

// SubscribeHandler handles subscription messages
type SubscribeHandler struct {
	subManager *subscription.Manager
	log        logger.Logger
}

// NewSubscribeHandler creates a new subscribe handler
func NewSubscribeHandler(subManager *subscription.Manager, log logger.Logger) *SubscribeHandler {
	return &SubscribeHandler{
		subManager: subManager,
		log:        log,
	}
}

// Handle handles a subscribe message
func (h *SubscribeHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.SubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Info("Processing subscription",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.Int("topic_count", len(payload.Topics)),
	)

	// Subscribe to topics
	for _, topic := range payload.Topics {
		if err := h.subManager.Subscribe(client, topic, payload.Filters); err != nil {
			h.log.Error("Failed to subscribe to topic",
				logger.String("client_id", client.ID),
				logger.String("topic", string(topic)),
				logger.Error(err),
			)
			// Continue with other topics
		}
	}

	// Send success response
	successMsg := protocol.ServerMessage{
		ID:        generateResponseID(msg.ID),
		Type:      protocol.ServerTypeSubscribed,
		RequestID: msg.ID,
		Payload: protocol.SubscribedPayload{
			Topics: payload.Topics,
		},
		Timestamp: time.Now(),
	}

	return client.SendMessage(successMsg)
}

// MessageType returns the message type this handler handles
func (h *SubscribeHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeSubscribe
}
