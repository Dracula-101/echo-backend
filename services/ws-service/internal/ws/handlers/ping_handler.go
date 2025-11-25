package handlers

import (
	"context"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

type PingHandler struct {
	log logger.Logger
}

func NewPingHandler(log logger.Logger) *PingHandler {
	return &PingHandler{log: log}
}

func (h *PingHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	h.log.Info("Processing ping message",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("message_id", msg.ID),
	)

	pongMsg := protocol.ServerMessage{
		ID:        "pong_" + msg.ID,
		Type:      protocol.ServerTypePong,
		RequestID: msg.ID,
		Payload: map[string]interface{}{
			"client_id":   client.ID,
			"server_time": time.Now().Unix(),
		},
		Timestamp: time.Now(),
	}

	h.log.Info("Sending pong response",
		logger.String("client_id", client.ID),
		logger.String("message_id", msg.ID),
		logger.String("pong_id", pongMsg.ID),
	)

	if err := client.SendMessage(pongMsg); err != nil {
		h.log.Error("Failed to send pong message",
			logger.String("client_id", client.ID),
			logger.String("message_id", msg.ID),
			logger.Error(err),
		)
		return err
	}

	h.log.Info("Pong message sent successfully",
		logger.String("client_id", client.ID),
		logger.String("message_id", msg.ID),
	)

	return nil
}

func (h *PingHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypePing
}
