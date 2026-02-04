package handler

import (
	"context"

	"shared/pkg/logger"
	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

type Handler interface {
	Handle(ctx context.Context, hctx *Context) error
}

type Context struct {
	Conn      Connection
	RequestID string
	Payload   []byte

	Deps *Dependencies
	Log  logger.Logger
}

type Dependencies struct {
	Subscriptions SubscriptionManager
	Presence      PresenceManager
	Typing        TypingManager
	Broadcaster   Broadcaster
	Authz         AuthorizationService
	Publisher     EventPublisher
}

type Connection interface {
	ID() string
	UserID() uuid.UUID
	DeviceID() string
	Platform() string
	Send(data []byte) error
	IsAuthenticated() bool
}

type SubscriptionManager interface {
	Subscribe(connID, topicKey string)
	Unsubscribe(connID, topicKey string)
	GetSubscriptions(connID string) []string
	GetSubscribers(topicKey string) []string
	CountSubscriptions(connID string) int
}

type PresenceManager interface {
	SetOnline(userID uuid.UUID, deviceID, platform string)
	SetOffline(userID uuid.UUID, deviceID string)
	UpdateStatus(userID uuid.UUID, status, customStatus string)
	GetPresence(userID uuid.UUID) *PresenceInfo
	GetBulkPresence(userIDs []uuid.UUID) map[uuid.UUID]*PresenceInfo
}

type PresenceInfo struct {
	UserID       uuid.UUID
	Status       string
	CustomStatus string
	DeviceCount  int
	LastSeenAt   string
}

type TypingManager interface {
	StartTyping(convID, userID uuid.UUID)
	StopTyping(convID, userID uuid.UUID)
	GetTypingUsers(convID uuid.UUID) []uuid.UUID
}

type Broadcaster interface {
	ToUser(userID uuid.UUID, msgType protocol.MessageType, payload any) error
	ToConversation(convID uuid.UUID, msgType string, payload any, excludeUserID ...uuid.UUID) error
	ToTopic(topic string, msgType string, payload any) error
}

type AuthorizationService interface {
	CanSubscribeToTopic(ctx context.Context, userID uuid.UUID, topic protocol.Topic, resourceID string) (bool, error)
}

type EventPublisher interface {
	PublishReadReceipt(ctx context.Context, event *ReadReceiptEvent) error
	PublishDeliveredReceipt(ctx context.Context, event *DeliveredReceiptEvent) error
}

type ReadReceiptEvent struct {
	UserID         uuid.UUID
	ConversationID uuid.UUID
	MessageIDs     []uuid.UUID
}

type DeliveredReceiptEvent struct {
	UserID         uuid.UUID
	ConversationID uuid.UUID
	MessageIDs     []uuid.UUID
}
