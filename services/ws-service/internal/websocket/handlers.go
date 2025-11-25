package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/server/websocket/connection"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

// Handler helper to get connection from metadata
func (m *Manager) getConnection(msg *router.Message) (*connection.Connection, bool) {
	connVal, ok := msg.Metadata["connection"]
	if !ok {
		return nil, false
	}
	conn, ok := connVal.(*connection.Connection)
	return conn, ok
}

// handleSubscribe handles subscription requests
func (m *Manager) handleSubscribe(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.SubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	for _, topic := range payload.Topics {
		resourceID := protocol.GetResourceID(topic, payload.Filters)
		topicKey := string(topic) + ":" + resourceID
		m.subscriptions.Subscribe(conn.ID(), topicKey)
	}

	// Send acknowledgment
	ack := protocol.ServerMessage{
		ID:        uuid.New().String(),
		Type:      "subscribed",
		Payload:   protocol.SubscribedPayload{Topics: payload.Topics},
		Timestamp: time.Now(),
		RequestID: msg.Metadata["message_id"].(string),
	}

	data, _ := json.Marshal(ack)
	return conn.Send(data)
}

// handleUnsubscribe handles unsubscribe requests
func (m *Manager) handleUnsubscribe(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.UnsubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	for _, topic := range payload.Topics {
		m.subscriptions.Unsubscribe(conn.ID(), string(topic))
	}

	// Send acknowledgment
	ack := protocol.ServerMessage{
		ID:        uuid.New().String(),
		Type:      "unsubscribed",
		Payload:   protocol.UnsubscribedPayload{Topics: payload.Topics},
		Timestamp: time.Now(),
		RequestID: msg.Metadata["message_id"].(string),
	}

	data, _ := json.Marshal(ack)
	return conn.Send(data)
}

// handlePresenceUpdate handles presence updates
func (m *Manager) handlePresenceUpdate(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userIDVal, _ := conn.GetMetadata("user_id")
	userID := userIDVal.(uuid.UUID)

	var payload protocol.PresenceUpdatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	m.presence.UpdatePresence(userID, PresenceStatus(payload.Status), payload.CustomStatus)

	// Broadcast presence update to contacts
	// TODO: Get user's contacts and broadcast
	return nil
}

// handlePresenceQuery handles presence queries
func (m *Manager) handlePresenceQuery(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.PresenceQueryPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	presences := m.presence.GetBulkPresence(payload.UserIDs)

	response := protocol.ServerMessage{
		ID:        uuid.New().String(),
		Type:      "presence.response",
		Payload:   presences,
		Timestamp: time.Now(),
		RequestID: msg.Metadata["message_id"].(string),
	}

	data, _ := json.Marshal(response)
	return conn.Send(data)
}

// handleTypingStart handles typing start events
func (m *Manager) handleTypingStart(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userIDVal, _ := conn.GetMetadata("user_id")
	userID := userIDVal.(uuid.UUID)

	var payload protocol.TypingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	m.typing.StartTyping(payload.ConversationID, userID)

	// Broadcast to conversation participants
	return m.BroadcastToConversation(payload.ConversationID, "typing.start",
		protocol.TypingEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			IsTyping:       true,
			Timestamp:      time.Now(),
		}, userID)
}

// handleTypingStop handles typing stop events
func (m *Manager) handleTypingStop(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userIDVal, _ := conn.GetMetadata("user_id")
	userID := userIDVal.(uuid.UUID)

	var payload protocol.TypingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	m.typing.StopTyping(payload.ConversationID, userID)

	// Broadcast to conversation participants
	return m.BroadcastToConversation(payload.ConversationID, "typing.stop",
		protocol.TypingEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			IsTyping:       false,
			Timestamp:      time.Now(),
		}, userID)
}

// handleMarkRead handles read receipt
func (m *Manager) handleMarkRead(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userIDVal, _ := conn.GetMetadata("user_id")
	userID := userIDVal.(uuid.UUID)

	var payload protocol.ReadReceiptPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	// Broadcast read receipt
	return m.BroadcastToConversation(payload.ConversationID, "message.read",
		protocol.ReadReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			Timestamp:      time.Now(),
		}, userID)
}

// handleMarkDelivered handles delivery receipt
func (m *Manager) handleMarkDelivered(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userIDVal, _ := conn.GetMetadata("user_id")
	userID := userIDVal.(uuid.UUID)

	var payload protocol.DeliveredReceiptPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	// Broadcast delivery receipt
	return m.BroadcastToConversation(payload.ConversationID, "message.delivered",
		protocol.DeliveredReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			Timestamp:      time.Now(),
		}, userID)
}

// handleCallOffer handles call offer
func (m *Manager) handleCallOffer(ctx context.Context, msg *router.Message) error {
	// TODO: Implement call signaling
	return nil
}

// handleCallAnswer handles call answer
func (m *Manager) handleCallAnswer(ctx context.Context, msg *router.Message) error {
	// TODO: Implement call signaling
	return nil
}

// handleCallICE handles ICE candidate
func (m *Manager) handleCallICE(ctx context.Context, msg *router.Message) error {
	// TODO: Implement call signaling
	return nil
}

// handleCallHangup handles call hangup
func (m *Manager) handleCallHangup(ctx context.Context, msg *router.Message) error {
	// TODO: Implement call signaling
	return nil
}

// handlePing handles ping message
func (m *Manager) handlePing(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	pong := protocol.ServerMessage{
		ID:        uuid.New().String(),
		Type:      "pong",
		Payload:   map[string]interface{}{"timestamp": time.Now()},
		Timestamp: time.Now(),
		RequestID: msg.Metadata["message_id"].(string),
	}

	data, _ := json.Marshal(pong)
	return conn.Send(data)
}
