package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
)

func (m *Manager) handlePing(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var pingPayload protocol.PingPayload
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, &pingPayload); err != nil {
			return err
		}
	}

	if ok := protocol.ValidatePingPayload(pingPayload); !ok {
		m.log.Warn("Invalid ping payload",
			logger.String("conn_id", conn.ID()),
		)
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeInvalidPayload, "Invalid ping payload")
	}

	serverTimestamp := time.Now().UTC()
	pongPayload := protocol.PongPayload{
		ClientTimestamp: pingPayload.ClientTimestamp,
		ServerTimestamp: serverTimestamp,
	}
	if !pingPayload.ClientTimestamp.IsZero() {
		pongPayload.Latency = serverTimestamp.Sub(pingPayload.ClientTimestamp).Milliseconds()
	}

	pong := protocol.NewServerMessage(protocol.MsgTypePong, m.getRequestID(msg)).WithPayload(pongPayload)

	data, _ := json.Marshal(pong)
	return conn.Send(data)
}
