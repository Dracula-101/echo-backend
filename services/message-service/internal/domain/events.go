package domain

import (
	"encoding/json"
	"time"

	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/utils"

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

func (e *ChatMessageEvent) ToDBModel() *dbModels.Message {
	var parentMessageID *string
	if e.ParentMessageID != nil {
		parentMessageID = utils.PtrString(e.ParentMessageID.String())
	}
	return &dbModels.Message{
		ID:                     e.ID.String(),
		ConversationID:         e.ConversationID.String(),
		SenderUserID:           e.SenderUserID.String(),
		ParentMessageID:        parentMessageID,
		MessageType:            dbModels.MessageType(e.MessageType),
		Content:                &e.Content,
		ContentEncrypted:       e.ContentEncrypted,
		ContentHash:            &e.ContentHash,
		FormatType:             dbModels.MessageFormatType(e.FormatType),
		Mentions:               e.Mentions,
		Hashtags:               e.Hashtags,
		Links:                  e.Links,
		Status:                 dbModels.MessageStatus(e.Status),
		DeliveredAt:            e.DeliveredAt,
		DeliveryCount:          e.DeliveryCount,
		ReadCount:              e.ReadCount,
		ReplyCount:             e.ReplyCount,
		LastReplyAt:            e.LastReplyAt,
		ReactionCount:          e.ReactionCount,
		IsForwarded:            e.IsForwarded,
		ForwardedFromMessageID: e.ForwardedFromMessageID,
		ForwardCount:           e.ForwardCount,
		SentFromDeviceID:       utils.PtrString(e.SentFromDeviceID),
		SentFromIP:             utils.PtrString(e.SentFromIP),
		CreatedAt:              e.CreatedAt,
		UpdatedAt:              e.UpdatedAt,
		Metadata:               e.Metadata,
	}
}

// ConversationEvent represents a conversatio
type ConversationEvent struct {
	EventType      ConversationEventType
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Timestamp      time.Time
	Payload        interface{}
}
