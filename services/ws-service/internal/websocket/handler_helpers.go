package websocket

import (
	"shared/server/websocket/connection"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

func (m *Manager) isSubscriptionAllowed(userID uuid.UUID, topic protocol.Topic, resourceID string) bool {
	switch topic {
	case protocol.TopicUser:
		return resourceID == userID.String()
	case protocol.TopicConversation:
		return true
	case protocol.TopicPresence:
		return true
	case protocol.TopicTyping:
		return true
	case protocol.TopicCalls:
		return true
	case protocol.TopicNotifications:
		return resourceID == userID.String()
	default:
		return false
	}
}

func (m *Manager) getConnection(msg *router.Message) (*connection.Connection, bool) {
	connVal, ok := msg.Metadata["connection"]
	if !ok {
		return nil, false
	}
	conn, ok := connVal.(*connection.Connection)
	return conn, ok
}

func (m *Manager) getRequestID(msg *router.Message) string {
	if id, ok := msg.Metadata["message_id"].(string); ok {
		return id
	}
	return ""
}
