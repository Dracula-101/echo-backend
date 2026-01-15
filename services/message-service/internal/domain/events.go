package domain

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Event Types (for producer/consumer model)
// ============================================================================

// MessageEvent represents a message-related event
type MessageEvent struct {
	EventType      MessageEventType
	MessageID      uuid.UUID
	ConversationID uuid.UUID
	SenderUserID   uuid.UUID
	RecipientIDs   []uuid.UUID
	Timestamp      time.Time
	Payload        interface{}
}

// ConversationEvent represents a conversation-related event
type ConversationEvent struct {
	EventType      ConversationEventType
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Timestamp      time.Time
	Payload        interface{}
}
