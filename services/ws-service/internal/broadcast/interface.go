package broadcast

import (
	"context"

	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

type Broadcaster interface {
	ToUser(ctx context.Context, userID uuid.UUID, msg *protocol.ServerMessage) error
	ToConversation(ctx context.Context, convID uuid.UUID, msg *protocol.ServerMessage, exclude ...uuid.UUID) error
	ToTopic(ctx context.Context, topic string, msg *protocol.ServerMessage) error
	ToConnection(ctx context.Context, connID string, msg *protocol.ServerMessage) error
}

type ConnectionProvider interface {
	GetConnection(connID string) (Connection, bool)
	GetUserConnections(userID uuid.UUID) []Connection
}

type SubscriptionProvider interface {
	GetSubscribers(topicKey string) []string
}

type Connection interface {
	ID() string
	UserID() uuid.UUID
	Send(data []byte) error
}

type Metrics interface {
	OnBroadcast()
	OnMessageSent(bytes int64)
}
