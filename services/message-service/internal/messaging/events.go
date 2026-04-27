package messaging

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MessageEventType string

const (
	MessageEventSent            MessageEventType = "message.sent"
	MessageEventDeleted         MessageEventType = "message.deleted"
	MessageEventEdited          MessageEventType = "message.edited"
	MessageEventReactionAdded   MessageEventType = "message.reaction_added"
	MessageEventReactionRemoved MessageEventType = "message.reaction_removed"
)

type messageEnvelope struct {
	EventID        uuid.UUID        `json:"event_id"`
	EventType      MessageEventType `json:"event_type"`
	UserID         uuid.UUID        `json:"user_id"`
	ConversationID uuid.UUID        `json:"conversation_id"`
	OccurredAt     time.Time        `json:"occurred_at"`
	Version        string           `json:"version"`
	Payload        json.RawMessage  `json:"payload"`
}
