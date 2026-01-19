package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/server/websocket/router"
	"ws-service/internal/protocol"
)

func (m *Manager) handleMarkRead(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.ReadReceiptPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	return m.BroadcastToConversation(payload.ConversationID, string(protocol.MsgTypeMessageRead),
		protocol.ReadReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			ReadAt:         time.Now(),
		}, userID)
}

func (m *Manager) handleMarkDelivered(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.DeliveredReceiptPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	return m.BroadcastToConversation(payload.ConversationID, string(protocol.MsgTypeMessageDelivered),
		protocol.DeliveredReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			DeliveredAt:    time.Now(),
		}, userID)
}
