package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
)

func (m *Manager) handleTypingStart(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.TypingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	if ok := protocol.ValidateTypingPayload(payload); !ok {
		m.log.Warn("Invalid typing payload",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", payload.ConversationID.String()),
		)
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeInvalidPayload, "Invalid typing payload")
	}

	m.typing.StartTyping(payload.ConversationID, userID)

	return m.BroadcastToConversation(payload.ConversationID, string(protocol.MsgTypeTypingStart),
		protocol.TypingEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			IsTyping:       true,
			Timestamp:      time.Now(),
		}, userID)
}

func (m *Manager) handleTypingStop(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.TypingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	if ok := protocol.ValidateTypingPayload(payload); !ok {
		m.log.Warn("Invalid typing payload",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", payload.ConversationID.String()),
		)
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeInvalidPayload, "Invalid typing payload")
	}

	m.typing.StopTyping(payload.ConversationID, userID)

	return m.BroadcastToConversation(payload.ConversationID, string(protocol.MsgTypeTypingStop),
		protocol.TypingEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			IsTyping:       false,
			Timestamp:      time.Now(),
		}, userID)
}
