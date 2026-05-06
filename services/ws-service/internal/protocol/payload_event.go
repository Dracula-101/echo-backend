package protocol

import (
	"time"

	"github.com/google/uuid"
)

type TypingEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool      `json:"is_typing"`
	Timestamp      time.Time `json:"timestamp"`
}

type ReadReceiptEvent struct {
	UserID         uuid.UUID   `json:"user_id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	ReadAt         time.Time   `json:"read_at"`
}

type DeliveredReceiptEvent struct {
	UserID         uuid.UUID   `json:"user_id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	DeliveredAt    time.Time   `json:"delivered_at"`
}

type PresenceChangedEvent struct {
	UserID       uuid.UUID `json:"user_id"`
	OldStatus    string    `json:"old_status,omitempty"`
	NewStatus    string    `json:"new_status"`
	CustomStatus string    `json:"custom_status,omitempty"`
	ChangedAt    time.Time `json:"changed_at"`
}

type ConnectionStatusEvent struct {
	Status      string    `json:"status"`
	Reason      string    `json:"reason,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	ResumeToken string    `json:"resume_token,omitempty"`
	InstanceID  string    `json:"instance_id,omitempty"`
}
