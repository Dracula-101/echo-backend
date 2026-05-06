package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/pubsub"
	"ws-service/internal/domain"
	"ws-service/internal/protocol"
	internalRedis "ws-service/internal/redis"
	"ws-service/websocket/tracker"

	"github.com/google/uuid"
)

func (m *Manager) BroadcastToUser(userID uuid.UUID, messageType string, payload any) error {
	data := m.marshalPayload(messageType, payload)

	// Local delivery via the in-process pubsub broker (subscribes registered with engine.PubSub).
	msg := &pubsub.Message{
		Topic:   "user:" + userID.String(),
		Payload: data,
		Metadata: map[string]any{
			"type":    messageType,
			"user_id": userID,
		},
	}
	m.engine.PubSub().Publish(msg)

	// Direct hub delivery — covers the case where a connection isn't subscribed to the user-topic.
	m.deliverToHubLocal(userID, data)

	m.relayUser(userID, data)

	if m.metrics != nil {
		m.metrics.OnBroadcast()
	}

	return nil
}

func (m *Manager) BroadcastToConversation(conversationID uuid.UUID, messageType string, payload any, excludeUserID ...uuid.UUID) error {
	data := m.marshalPayload(messageType, payload)

	m.deliverToConversationLocal(conversationID, data, excludeUserID)

	m.relayConversation(conversationID, data, excludeUserID)

	if m.metrics != nil {
		m.metrics.OnBroadcast()
	}

	return nil
}

// deliverToConversationLocal sends to subscribers connected to THIS instance only.
// Pulled out so cross-instance relays can reuse it without re-publishing.
func (m *Manager) deliverToConversationLocal(conversationID uuid.UUID, data []byte, excludeUserID []uuid.UUID) {
	topicKey := "conversation:" + conversationID.String()
	subscribers := m.subscriptions.GetSubscribers(topicKey)

	m.log.Debug("Broadcasting to conversation",
		logger.String("topic_key", topicKey),
		logger.Int("subscriber_count", len(subscribers)),
	)

	sentCount := 0
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

		conn.Send(data)
		sentCount++
	}

	m.log.Debug("Broadcast completed",
		logger.String("topic_key", topicKey),
		logger.Int("sent_count", sentCount),
	)
}

// deliverToHubLocal sends to all connections of a user on THIS instance.
func (m *Manager) deliverToHubLocal(userID uuid.UUID, data []byte) {
	client, ok := m.hub.GetClient(userID)
	if !ok {
		return
	}
	for _, conn := range client.GetAllConnections() {
		conn.Send(data)
	}
}

// relayConversation publishes to the cross-instance pub/sub channel.
// No-op when pubsub isn't wired (single-instance mode).
func (m *Manager) relayConversation(conversationID uuid.UUID, data []byte, excludeUserID []uuid.UUID) {
	if m.pubsub == nil {
		return
	}

	excludeIDs := make([]string, len(excludeUserID))
	for i, id := range excludeUserID {
		excludeIDs[i] = id.String()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.pubsub.PublishBroadcast(ctx, &internalRedis.BroadcastMessage{
		Type:       "conversation",
		TargetType: "conversation",
		TargetID:   conversationID.String(),
		ExcludeIDs: excludeIDs,
		Payload:    data,
	}); err != nil {
		m.log.Warn("cross-instance conversation publish failed",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
	}
}

// relayUser publishes a user-targeted message to other instances.
func (m *Manager) relayUser(userID uuid.UUID, data []byte) {
	if m.pubsub == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.pubsub.PublishBroadcast(ctx, &internalRedis.BroadcastMessage{
		Type:       "user",
		TargetType: "user",
		TargetID:   userID.String(),
		Payload:    data,
	}); err != nil {
		m.log.Warn("cross-instance user publish failed",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	}
}

// handleRelayBroadcast is called by the Redis pubsub listener when ANOTHER
// instance has published a message. Self-instance messages are filtered by
// PubSub.handleMessage before this is invoked.
func (m *Manager) handleRelayBroadcast(msg *internalRedis.BroadcastMessage) {
	switch msg.TargetType {
	case "conversation":
		convID, err := uuid.Parse(msg.TargetID)
		if err != nil {
			return
		}
		exclude := make([]uuid.UUID, 0, len(msg.ExcludeIDs))
		for _, idStr := range msg.ExcludeIDs {
			if id, parseErr := uuid.Parse(idStr); parseErr == nil {
				exclude = append(exclude, id)
			}
		}
		m.deliverToConversationLocal(convID, msg.Payload, exclude)
	case "user":
		userID, err := uuid.Parse(msg.TargetID)
		if err != nil {
			return
		}
		// session.revoked frames carry their own payload type. Delivering it
		// locally also signals the connection to close — see RevokeUserSessions.
		if isSessionRevokeFrame(msg.Payload) {
			m.closeAllUserConnections(userID, msg.Payload)
			return
		}
		m.deliverToHubLocal(userID, msg.Payload)
	}
}

// closeAllUserConnections sends the frame and immediately closes every active
// connection for the user. Used by the cross-instance relay path of
// session-revocation events.
func (m *Manager) closeAllUserConnections(userID uuid.UUID, frame []byte) {
	client, ok := m.hub.GetClient(userID)
	if !ok {
		return
	}
	for _, conn := range client.GetAllConnections() {
		conn.Send(frame)
		conn.Close()
	}
}

// isSessionRevokeFrame inspects the marshalled JSON envelope and returns true
// when the type field is "session.revoked".
func isSessionRevokeFrame(frame []byte) bool {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(frame, &probe); err != nil {
		return false
	}
	return probe.Type == string(protocol.MsgTypeSessionRevoked)
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

	m.deliverToHubLocal(userID, data)
	m.relayUser(userID, data)
	m.bufferIfOffline(userID, data)

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

// SendEventToUser sends an event with a specific message type directly to a user's connections
func (m *Manager) SendEventToUser(userID uuid.UUID, messageType protocol.MessageType, payload any) error {
	msg := protocol.NewServerMessage(messageType, "").
		WithPayload(payload)

	return m.SendMessageToUser(userID, msg)
}
