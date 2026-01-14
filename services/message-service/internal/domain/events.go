package domain

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Event Types (for event-driven architecture)
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
