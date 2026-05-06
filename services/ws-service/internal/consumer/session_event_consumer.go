package consumer

import (
	"context"
	"encoding/json"

	"shared/pkg/logger"
	"shared/pkg/messaging"
	"ws-service/websocket"

	"github.com/google/uuid"
)

// SessionEventConsumer reacts to auth-service Kafka events. The only event we
// care about today is auth.session.revoked — when received, we look up any
// active local connections for the affected user and close them with a
// session.revoked frame.
//
// Session-scoped vs user-scoped: the outbox payload doesn't currently carry
// device_id, so we close ALL of the user's connections. When auth-service
// extends the payload, this can be narrowed.
type SessionEventConsumer struct {
	manager *websocket.Manager
	log     logger.Logger
}

const (
	EventTypeSessionRevoked = "auth.session.revoked"
)

type sessionRevokedPayload struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

func NewSessionEventConsumer(manager *websocket.Manager, log logger.Logger) *SessionEventConsumer {
	return &SessionEventConsumer{manager: manager, log: log}
}

func (c *SessionEventConsumer) Handle(ctx context.Context, message *messaging.Message) error {
	eventType := message.Headers["event_type"]
	if eventType != EventTypeSessionRevoked {
		return nil
	}

	var payload sessionRevokedPayload
	if err := json.Unmarshal(message.Value, &payload); err != nil {
		c.log.Warn("malformed session-revoked payload",
			logger.String("topic", message.Topic),
			logger.Error(err),
		)
		return nil
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return nil
	}

	c.manager.RevokeUserSessions(userID, payload.SessionID, "force_logout_session_revoked")
	return nil
}
