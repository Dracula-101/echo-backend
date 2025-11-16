package model

import (
	"time"

	"github.com/google/uuid"
)

// EventCategory represents the category of real-time event
type EventCategory string

const (
	CategoryPresence     EventCategory = "presence"
	CategoryMessaging    EventCategory = "messaging"
	CategoryTyping       EventCategory = "typing"
	CategoryCall         EventCategory = "call"
	CategoryNotification EventCategory = "notification"
	CategoryUser         EventCategory = "user"
	CategorySystem       EventCategory = "system"
)

// EventType represents specific event types within categories
type EventType string

// Presence Event Types
const (
	EventPresenceOnline    EventType = "presence.online"
	EventPresenceOffline   EventType = "presence.offline"
	EventPresenceAway      EventType = "presence.away"
	EventPresenceBusy      EventType = "presence.busy"
	EventPresenceInvisible EventType = "presence.invisible"
	EventPresenceUpdate    EventType = "presence.update"
)

// Messaging Event Types
const (
	EventMessageNew      EventType = "message.new"
	EventMessageDelivered EventType = "message.delivered"
	EventMessageRead     EventType = "message.read"
	EventMessageEdited   EventType = "message.edited"
	EventMessageDeleted  EventType = "message.deleted"
)

// Typing Event Types
const (
	EventTypingStart EventType = "typing.start"
	EventTypingStop  EventType = "typing.stop"
)

// Call Event Types
const (
	EventCallIncoming EventType = "call.incoming"
	EventCallAccepted EventType = "call.accepted"
	EventCallRejected EventType = "call.rejected"
	EventCallEnded    EventType = "call.ended"
	EventCallMissed   EventType = "call.missed"
)

// Notification Event Types
const (
	EventNotificationNew EventType = "notification.new"
	EventNotificationRead EventType = "notification.read"
)

// User Event Types
const (
	EventUserProfileUpdated EventType = "user.profile_updated"
	EventUserStatusUpdated  EventType = "user.status_updated"
	EventUserBlocked        EventType = "user.blocked"
	EventUserUnblocked      EventType = "user.unblocked"
)

// System Event Types
const (
	EventSystemMaintenance  EventType = "system.maintenance"
	EventSystemAnnouncement EventType = "system.announcement"
)

// RealtimeEvent represents a real-time event to be broadcast
type RealtimeEvent struct {
	// Event metadata
	ID        uuid.UUID     `json:"id"`
	Type      EventType     `json:"type"`
	Category  EventCategory `json:"category"`
	Timestamp time.Time     `json:"timestamp"`

	// Routing information
	Recipients []uuid.UUID `json:"recipients"` // User IDs to receive this event
	Sender     *uuid.UUID  `json:"sender,omitempty"` // User ID of sender (if applicable)

	// Event payload (specific to event type)
	Payload interface{} `json:"payload"`

	// Optional metadata
	Priority  int    `json:"priority,omitempty"` // 0=normal, 1=high, 2=urgent
	TTL       int    `json:"ttl,omitempty"`      // Time to live in seconds (for offline storage)
}

// BroadcastRequest represents a request from other services to broadcast events
type BroadcastRequest struct {
	EventType  EventType   `json:"event_type" validate:"required"`
	Recipients []uuid.UUID `json:"recipients" validate:"required,min=1"`
	Sender     *uuid.UUID  `json:"sender,omitempty"`
	Payload    interface{} `json:"payload" validate:"required"`
	Priority   int         `json:"priority,omitempty"`
	TTL        int         `json:"ttl,omitempty"`
}

// Specific Event Payloads

// PresencePayload for presence events
type PresencePayload struct {
	UserID       uuid.UUID  `json:"user_id"`
	OnlineStatus string     `json:"online_status"`
	CustomStatus string     `json:"custom_status,omitempty"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
}

// MessagePayload for message events
type MessagePayload struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Content        string    `json:"content,omitempty"`
	MessageType    string    `json:"message_type,omitempty"` // text, image, video, etc.
	Timestamp      time.Time `json:"timestamp"`
}

// TypingPayload for typing events
type TypingPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	IsTyping       bool      `json:"is_typing"`
}

// CallPayload for call events
type CallPayload struct {
	CallID     uuid.UUID `json:"call_id"`
	CallerID   uuid.UUID `json:"caller_id"`
	CalleeID   uuid.UUID `json:"callee_id"`
	CallType   string    `json:"call_type"` // audio, video
	RoomID     string    `json:"room_id,omitempty"`
	SignalData string    `json:"signal_data,omitempty"` // WebRTC signaling data
}

// NotificationPayload for notification events
type NotificationPayload struct {
	NotificationID uuid.UUID              `json:"notification_id"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	Data           map[string]interface{} `json:"data,omitempty"`
	ActionURL      string                 `json:"action_url,omitempty"`
}

// UserPayload for user events
type UserPayload struct {
	UserID  uuid.UUID              `json:"user_id"`
	Updates map[string]interface{} `json:"updates"`
}

// SystemPayload for system events
type SystemPayload struct {
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
