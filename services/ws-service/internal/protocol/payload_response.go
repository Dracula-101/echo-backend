package protocol

import (
	"time"

	"github.com/google/uuid"
)

type SubscribedPayload struct {
	Topics            []Topic              `json:"topics"`
	SubscriptionCount int                  `json:"subscription_count"`
	Subscriptions     []SubscriptionDetail `json:"subscriptions,omitempty"`
}

type SubscriptionDetail struct {
	Topic      Topic  `json:"topic"`
	ResourceID string `json:"resource_id"`
	Active     bool   `json:"active"`
}

type UnsubscribedPayload struct {
	Topics             []Topic `json:"topics"`
	RemainingCount     int     `json:"remaining_count"`
	UnsubscribedTopics []Topic `json:"unsubscribed_topics"`
}

type PresenceResponsePayload struct {
	Presences    []PresenceInfo `json:"presences"`
	OfflineUsers []uuid.UUID    `json:"offline_users,omitempty"`
}

type PresenceInfo struct {
	UserID       uuid.UUID `json:"user_id"`
	Status       string    `json:"status"`
	CustomStatus string    `json:"custom_status,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	DeviceCount  int       `json:"device_count"`
}

type PongPayload struct {
	ClientTimestamp time.Time `json:"client_timestamp,omitempty"`
	ServerTimestamp time.Time `json:"server_timestamp"`
	Latency         int64     `json:"latency_ms,omitempty"`
}
