package domain

import (
	"time"

	"github.com/google/uuid"
)

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

type EventType string

const (
	EventPresenceOnline    EventType = "presence.online"
	EventPresenceOffline   EventType = "presence.offline"
	EventPresenceAway      EventType = "presence.away"
	EventPresenceBusy      EventType = "presence.busy"
	EventPresenceInvisible EventType = "presence.invisible"
	EventPresenceUpdate    EventType = "presence.update"
)

const (
	EventMessageNew       EventType = "message.new"
	EventMessageDelivered EventType = "message.delivered"
	EventMessageRead      EventType = "message.read"
	EventMessageEdited    EventType = "message.edited"
	EventMessageDeleted   EventType = "message.deleted"
)

const (
	EventTypingStart EventType = "typing.start"
	EventTypingStop  EventType = "typing.stop"
)

const (
	EventCallIncoming EventType = "call.incoming"
	EventCallAccepted EventType = "call.accepted"
	EventCallRejected EventType = "call.rejected"
	EventCallEnded    EventType = "call.ended"
	EventCallMissed   EventType = "call.missed"
)

const (
	EventNotificationNew  EventType = "notification.new"
	EventNotificationRead EventType = "notification.read"
)

const (
	EventUserProfileUpdated EventType = "user.profile_updated"
	EventUserStatusUpdated  EventType = "user.status_updated"
	EventUserBlocked        EventType = "user.blocked"
	EventUserUnblocked      EventType = "user.unblocked"
)

const (
	EventSystemMaintenance  EventType = "system.maintenance"
	EventSystemAnnouncement EventType = "system.announcement"
)

type RealtimeEvent struct {
	ID         uuid.UUID     `json:"id"`
	Type       EventType     `json:"type"`
	Category   EventCategory `json:"category"`
	Timestamp  time.Time     `json:"timestamp"`
	Recipients []uuid.UUID   `json:"recipients"`
	Sender     *uuid.UUID    `json:"sender,omitempty"`
	Payload    interface{}   `json:"payload"`
	Priority   int           `json:"priority,omitempty"`
	TTL        int           `json:"ttl,omitempty"`
}

type BroadcastRequest struct {
	EventType  EventType   `json:"event_type" validate:"required"`
	Recipients []uuid.UUID `json:"recipients" validate:"required,min=1"`
	Sender     *uuid.UUID  `json:"sender,omitempty"`
	Payload    interface{} `json:"payload" validate:"required"`
	Priority   int         `json:"priority,omitempty"`
	TTL        int         `json:"ttl,omitempty"`
}

type BroadcastResponse struct {
	EventID          uuid.UUID `json:"event_id"`
	Recipients       int       `json:"recipients"`
	OnlineRecipients int       `json:"online_recipients"`
	Timestamp        time.Time `json:"timestamp"`
}
