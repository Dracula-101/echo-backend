package handler

import (
	"context"

	"shared/server/websocket/connection"
	"shared/server/websocket/router"

	"github.com/google/uuid"
)

type MessageHandler func(ctx context.Context, msg *router.Message) error

type Dependencies struct {
	GetConnection      func(connID string) (*connection.Connection, bool)
	GetUserID          func(conn *connection.Connection) (uuid.UUID, bool)
	GetDeviceID        func(conn *connection.Connection) (string, bool)
	Subscribe          func(connID, topic string) bool
	Unsubscribe        func(connID, topic string)
	GetSubscriptions   func(connID string) []string
	CountSubscriptions func(connID string) int
	IsAllowedTopic     func(userID uuid.UUID, topic string, resourceID string) bool
	StartTyping        func(conversationID, userID uuid.UUID)
	StopTyping         func(conversationID, userID uuid.UUID)
	UpdatePresence     func(userID uuid.UUID, status string, customStatus string)
	GetPresence        func(userID uuid.UUID) PresenceResult
	GetBulkPresence    func(userIDs []uuid.UUID) map[uuid.UUID]PresenceResult
	BroadcastToConv    func(conversationID uuid.UUID, msgType string, payload any, excludeUser ...uuid.UUID) error
	SendResponse       func(conn *connection.Connection, requestID string, msgType string, payload any) error
	SendError          func(conn *connection.Connection, requestID string, code string, message string) error
	RecordMetric       func(msgType string, sizeBytes int64)
	RecordLatency      func(latencyNs int64)
}

type PresenceResult struct {
	UserID       uuid.UUID `json:"user_id"`
	Status       string    `json:"status"`
	CustomStatus string    `json:"custom_status,omitempty"`
	LastSeenAt   int64     `json:"last_seen_at"`
	DeviceCount  int       `json:"device_count"`
	Online       bool      `json:"online"`
}

type HandlerContext struct {
	Connection *connection.Connection
	UserID     uuid.UUID
	DeviceID   string
	RequestID  string
}

func ExtractContext(msg *router.Message, deps *Dependencies) (*HandlerContext, bool) {
	connVal, ok := msg.Metadata["connection"]
	if !ok {
		return nil, false
	}

	conn, ok := connVal.(*connection.Connection)
	if !ok {
		return nil, false
	}

	userID, ok := deps.GetUserID(conn)
	if !ok {
		return nil, false
	}

	deviceID, _ := deps.GetDeviceID(conn)

	requestID := ""
	if id, ok := msg.Metadata["message_id"].(string); ok {
		requestID = id
	}

	return &HandlerContext{
		Connection: conn,
		UserID:     userID,
		DeviceID:   deviceID,
		RequestID:  requestID,
	}, true
}
