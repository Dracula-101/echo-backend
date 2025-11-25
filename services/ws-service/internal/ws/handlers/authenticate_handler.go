package handlers

import (
	"context"
	"encoding/json"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

// AuthenticateHandler handles authentication messages
type AuthenticateHandler struct {
	log logger.Logger
}

// NewAuthenticateHandler creates a new authenticate handler
func NewAuthenticateHandler(log logger.Logger) *AuthenticateHandler {
	return &AuthenticateHandler{log: log}
}

// Handle handles an authenticate message
func (h *AuthenticateHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.AuthenticatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return h.sendAuthFailed(client, msg.ID, "Invalid payload")
	}

	h.log.Info("Processing authentication",
		logger.String("client_id", client.ID),
		logger.String("device_id", payload.DeviceID),
		logger.String("platform", payload.Platform),
	)

	// TODO: Validate token with auth service
	// For now, accept all tokens

	// Send success response
	successMsg := protocol.ServerMessage{
		ID:        generateResponseID(msg.ID),
		Type:      protocol.ServerTypeAuthSuccess,
		RequestID: msg.ID,
		Payload: map[string]interface{}{
			"authenticated": true,
			"user_id":       client.UserID.String(),
			"device_id":     payload.DeviceID,
			"platform":      payload.Platform,
		},
		Timestamp: time.Now(),
	}

	return client.SendMessage(successMsg)
}

// MessageType returns the message type this handler handles
func (h *AuthenticateHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeAuthenticate
}

func (h *AuthenticateHandler) sendAuthFailed(client *websocket.Client, requestID, reason string) error {
	failMsg := protocol.ServerMessage{
		ID:        generateResponseID(requestID),
		Type:      protocol.ServerTypeAuthFailed,
		RequestID: requestID,
		Payload: protocol.ErrorPayload{
			Code:    "auth_failed",
			Message: reason,
		},
		Timestamp: time.Now(),
	}

	return client.SendMessage(failMsg)
}
