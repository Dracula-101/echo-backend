package model

import (
	"time"

	"github.com/google/uuid"
)

// UserPresence represents the current presence status of a user
type UserPresence struct {
	UserID       uuid.UUID  `json:"user_id"`
	OnlineStatus string     `json:"online_status"` // online, offline, away, busy, invisible
	LastSeenAt   *time.Time `json:"last_seen_at"`
	CustomStatus string     `json:"custom_status,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Device represents a user's device information
type Device struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	DeviceID       string     `json:"device_id"`
	DeviceName     string     `json:"device_name,omitempty"`
	DeviceType     string     `json:"device_type,omitempty"`
	Platform       string     `json:"platform,omitempty"`
	AppVersion     string     `json:"app_version,omitempty"`
	IsActive       bool       `json:"is_active"`
	LastActiveAt   time.Time  `json:"last_active_at"`
	RegisteredAt   time.Time  `json:"registered_at"`
	FCMToken       string     `json:"fcm_token,omitempty"`
	APNSToken      string     `json:"apns_token,omitempty"`
	PushEnabled    bool       `json:"push_enabled"`
}

// PresenceUpdate represents a presence status update
type PresenceUpdate struct {
	UserID       uuid.UUID `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	OnlineStatus string    `json:"online_status"`
	CustomStatus string    `json:"custom_status,omitempty"`
}

// TypingIndicator represents typing status in a conversation
type TypingIndicator struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	DeviceID       string    `json:"device_id,omitempty"`
	IsTyping       bool      `json:"is_typing"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PresencePrivacy represents privacy settings for presence
type PresencePrivacy struct {
	UserID                  uuid.UUID `json:"user_id"`
	LastSeenVisibility      string    `json:"last_seen_visibility"`      // everyone, contacts, nobody
	OnlineStatusVisibility  string    `json:"online_status_visibility"`  // everyone, contacts, nobody
	TypingIndicatorsEnabled bool      `json:"typing_indicators_enabled"`
	ReadReceiptsEnabled     bool      `json:"read_receipts_enabled"`
}

// BulkPresenceRequest represents a request for multiple users' presence
type BulkPresenceRequest struct {
	UserIDs []uuid.UUID `json:"user_ids"`
}

// BulkPresenceResponse represents presence info for multiple users
type BulkPresenceResponse struct {
	Presences map[uuid.UUID]UserPresence `json:"presences"`
}
