package state

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SubscriptionStore interface {
	Subscribe(ctx context.Context, connID, topicKey string) error
	Unsubscribe(ctx context.Context, connID, topicKey string) error
	UnsubscribeAll(ctx context.Context, connID string) ([]string, error)
	GetSubscribers(ctx context.Context, topicKey string) ([]string, error)
	GetSubscriptions(ctx context.Context, connID string) ([]string, error)
	IsSubscribed(ctx context.Context, connID, topicKey string) (bool, error)
	CountSubscriptions(ctx context.Context, connID string) (int, error)
}

type PresenceInfo struct {
	UserID       uuid.UUID
	Status       string
	CustomStatus string
	Devices      []DeviceInfo
	LastSeenAt   time.Time
}

type DeviceInfo struct {
	DeviceID    string
	Platform    string
	ConnID      string
	ConnectedAt time.Time
}

type PresenceStore interface {
	SetOnline(ctx context.Context, userID uuid.UUID, deviceID, platform, connID string) error
	SetOffline(ctx context.Context, userID uuid.UUID, deviceID, connID string) (isLastConnection bool, err error)
	UpdateStatus(ctx context.Context, userID uuid.UUID, status, customStatus string) error
	GetPresence(ctx context.Context, userID uuid.UUID) (*PresenceInfo, error)
	GetPresenceBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*PresenceInfo, error)
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
}

type TypingStore interface {
	StartTyping(ctx context.Context, convID, userID uuid.UUID) error
	StopTyping(ctx context.Context, convID, userID uuid.UUID) error
	GetTypingUsers(ctx context.Context, convID uuid.UUID) ([]uuid.UUID, error)
}

type ConnectionInfo struct {
	ConnID      string
	UserID      uuid.UUID
	DeviceID    string
	Platform    string
	InstanceID  string
	ConnectedAt time.Time
}

type ConnectionRegistry interface {
	Register(ctx context.Context, info *ConnectionInfo) error
	Unregister(ctx context.Context, connID string) ([]string, error)
	GetConnection(ctx context.Context, connID string) (*ConnectionInfo, error)
	GetUserConnections(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetInstanceConnections(ctx context.Context, instanceID string) ([]string, error)
	RefreshTTL(ctx context.Context, connID string) error
}
