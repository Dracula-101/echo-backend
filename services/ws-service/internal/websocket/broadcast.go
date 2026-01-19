package websocket

import (
	"encoding/json"
	"time"

	"shared/server/websocket/pubsub"
	"ws-service/internal/domain"
	"ws-service/internal/protocol"
	"ws-service/internal/websocket/tracker"

	"github.com/google/uuid"
)

func (m *Manager) BroadcastToUser(userID uuid.UUID, messageType string, payload any) error {
	msg := &pubsub.Message{
		Topic:   "user:" + userID.String(),
		Payload: m.marshalPayload(messageType, payload),
		Metadata: map[string]any{
			"type":    messageType,
			"user_id": userID,
		},
	}

	m.engine.PubSub().Publish(msg)

	if m.metrics != nil {
		m.metrics.OnBroadcast()
	}

	return nil
}

func (m *Manager) BroadcastToConversation(conversationID uuid.UUID, messageType string, payload any, excludeUserID ...uuid.UUID) error {
	subscribers := m.subscriptions.GetSubscribers("conversation:" + conversationID.String())

	for _, connID := range subscribers {
		conn, ok := m.engine.ConnectionManager().Get(connID)
		if !ok {
			continue
		}

		userIDVal, ok := conn.GetMetadata(domain.UserIdMetaKey)
		if !ok {
			continue
		}

		userID := userIDVal.(uuid.UUID)
		skip := false
		for _, excludeID := range excludeUserID {
			if userID == excludeID {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		data := m.marshalPayload(messageType, payload)
		conn.Send(data)
	}

	if m.metrics != nil {
		m.metrics.OnBroadcast()
	}

	return nil
}

func (m *Manager) BroadcastPresenceChange(event tracker.PresenceChangeEvent) {
	payload := protocol.PresenceChangedEvent{
		UserID:       event.UserID,
		OldStatus:    string(event.OldStatus),
		NewStatus:    string(event.NewStatus),
		CustomStatus: event.CustomStatus,
		ChangedAt:    event.ChangedAt,
	}

	subscribers := m.subscriptions.GetSubscribers("presence:global")
	for _, connID := range subscribers {
		conn, ok := m.engine.ConnectionManager().Get(connID)
		if !ok {
			continue
		}

		m.sendResponse(conn, "", protocol.MsgTypePresenceChanged, payload)
	}

	if m.metrics != nil {
		m.metrics.OnBroadcast()
	}
}

func (m *Manager) BroadcastTypingEvent(event tracker.TypingEvent) {
	payload := protocol.TypingEvent{
		UserID:         event.UserID,
		ConversationID: event.ConversationID,
		IsTyping:       event.IsTyping,
		Timestamp:      event.Timestamp,
	}

	msgType := protocol.MsgTypeTypingStart
	if !event.IsTyping {
		msgType = protocol.MsgTypeTypingStop
	}

	m.BroadcastToConversation(event.ConversationID, string(msgType), payload, event.UserID)
}

func (m *Manager) BroadcastToTopic(topic string, messageType string, payload any) error {
	subscribers := m.subscriptions.GetSubscribers(topic)

	for _, connID := range subscribers {
		conn, ok := m.engine.ConnectionManager().Get(connID)
		if !ok {
			continue
		}

		data := m.marshalPayload(messageType, payload)
		conn.Send(data)
	}

	if m.metrics != nil {
		m.metrics.OnBroadcast()
	}

	return nil
}

func (m *Manager) marshalPayload(messageType string, payload any) []byte {
	msg := protocol.NewServerMessage(protocol.MessageType(messageType), "").WithPayload(payload)
	data, _ := json.Marshal(msg)
	return data
}

func (m *Manager) SendToConnection(connID string, messageType string, payload any) error {
	conn, ok := m.engine.ConnectionManager().Get(connID)
	if !ok {
		return nil
	}

	data := m.marshalPayload(messageType, payload)
	return conn.Send(data)
}

func (m *Manager) SendMessageToUser(userID uuid.UUID, msg *protocol.ServerMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	client, ok := m.hub.GetClient(userID)
	if !ok {
		return nil
	}

	conns := client.GetAllConnections()
	for _, conn := range conns {
		conn.Send(data)
	}

	if m.metrics != nil {
		m.metrics.OnMessageSent(int64(len(data)))
	}

	return nil
}

func (m *Manager) SendNewMessageEvent(userID uuid.UUID, event any) error {
	msg := protocol.NewServerMessage(protocol.MsgTypeMessageNew, "").
		WithPayload(event).
		WithMetadata(m.config.MetricsPrefix, time.Since(m.startTime).Milliseconds())

	return m.SendMessageToUser(userID, msg)
}
