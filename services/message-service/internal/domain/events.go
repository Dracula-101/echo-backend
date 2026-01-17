package domain

import (
	"encoding/json"
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

type MessageFormatType string

const (
	MessageFormatPlain    MessageFormatType = "plain"
	MessageFormatMarkdown MessageFormatType = "markdown"
	MessageFormatHTML     MessageFormatType = "html"
)

type ChatMessageEvent struct {
	ID                     uuid.UUID
	ConversationID         uuid.UUID
	SenderUserID           uuid.UUID
	ParentMessageID        *uuid.UUID
	MessageType            MessageType
	Content                string
	ContentEncrypted       bool
	ContentHash            string
	FormatType             MessageFormatType
	Mentions               json.RawMessage
	Hashtags               []string
	Links                  json.RawMessage
	Status                 MessageStatus
	DeliveredAt            *time.Time
	DeliveryCount          int
	ReadCount              int
	ReplyCount             int
	LastReplyAt            *time.Time
	ReactionCount          int
	IsForwarded            bool
	ForwardedFromMessageID *string
	ForwardCount           int
	SentFromDeviceID       string
	SentFromIP             string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Metadata               json.RawMessage
}

// ConversationEvent represents a conversatio
type ConversationEvent struct {
	EventType      ConversationEventType
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Timestamp      time.Time
	Payload        interface{}
}
