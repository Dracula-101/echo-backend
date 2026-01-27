package websocket

import (
	"shared/server/websocket/connection"
	"shared/server/websocket/router"
)

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
