package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
	"ws-service/internal/service"
)

func (m *Manager) handleMarkRead(_ context.Context, msg *router.Message) error {
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

	now := time.Now()

	// Publish delivery event to Kafka for persistence
	if m.deliveryPublisher != nil {
		// Use detached context to avoid cancellation from WebSocket
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		deliveryEvent := &service.DeliveryEvent{
			EventType:      service.DeliveryEventRead,
			MessageIDs:     payload.MessageIDs,
			UserID:         userID,
			ConversationID: payload.ConversationID,
			Timestamp:      now,
		}
		if err := m.deliveryPublisher.PublishDeliveryEvent(publishCtx, deliveryEvent); err != nil {
			m.log.Error("Failed to publish read receipt event",
				logger.String("user_id", userID.String()),
				logger.String("conversation_id", payload.ConversationID.String()),
				logger.Error(err),
			)
			// Continue with broadcast even if publish fails
		}
	}

	// Broadcast to conversation participants
	return m.BroadcastToConversation(payload.ConversationID, string(protocol.MsgTypeMessageRead),
		protocol.ReadReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			ReadAt:         now,
		}, userID)
}

func (m *Manager) handleMarkDelivered(_ context.Context, msg *router.Message) error {
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

	now := time.Now()

	// Publish delivery event to Kafka for persistence
	if m.deliveryPublisher != nil {
		// Use detached context to avoid cancellation from WebSocket
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		deliveryEvent := &service.DeliveryEvent{
			EventType:      service.DeliveryEventDelivered,
			MessageIDs:     payload.MessageIDs,
			UserID:         userID,
			ConversationID: payload.ConversationID,
			Timestamp:      now,
		}
		if err := m.deliveryPublisher.PublishDeliveryEvent(publishCtx, deliveryEvent); err != nil {
			m.log.Error("Failed to publish delivered receipt event",
				logger.String("user_id", userID.String()),
				logger.String("conversation_id", payload.ConversationID.String()),
				logger.Error(err),
			)
			// Continue with broadcast even if publish fails
		}
	}

	// Broadcast to read conversation participants
	return m.BroadcastToConversation(payload.ConversationID, string(protocol.MsgTypeMarkDelivered),
		protocol.DeliveredReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			DeliveredAt:    now,
		}, userID)
}
