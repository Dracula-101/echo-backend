package connection

import (
	"github.com/google/uuid"
)

type Manager interface {
	OnConnect(conn Connection) error
	OnDisconnect(conn Connection) error
	GetConnection(connID string) (Connection, bool)
	GetUserConnections(userID uuid.UUID) []Connection
	SendToUser(userID uuid.UUID, data []byte) error
}

type Connection interface {
	ID() string
	UserID() uuid.UUID
	DeviceID() string
	Platform() string
	Send(data []byte) error
	Close() error
	IsAuthenticated() bool
	SetMetadata(key string, value any)
	GetMetadata(key string) (any, bool)
}

type Hook func(conn Connection)

type Hooks interface {
	AddOnConnect(fn Hook)
	AddOnDisconnect(fn Hook)
}

type PresenceNotifier interface {
	SetOnline(userID uuid.UUID, deviceID, platform string) error
	SetOffline(userID uuid.UUID, deviceID string) error
}

type SubscriptionCleaner interface {
	UnsubscribeAll(connID string) ([]string, error)
}
