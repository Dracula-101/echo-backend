package websocket

import (
	"context"

	"shared/server/websocket/router"
	"ws-service/internal/protocol"
)

func (m *Manager) handleCallOffer(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}
	return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeNotImplemented, "Call signaling is not yet implemented")
}

func (m *Manager) handleCallAnswer(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}
	return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeNotImplemented, "Call signaling is not yet implemented")
}

func (m *Manager) handleCallICE(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}
	return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeNotImplemented, "Call signaling is not yet implemented")
}

func (m *Manager) handleCallHangup(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}
	return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeNotImplemented, "Call signaling is not yet implemented")
}
