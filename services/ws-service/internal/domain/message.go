package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of message content
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeFile     MessageType = "file"
	MessageTypeLocation MessageType = "location"
)

// MessageStatus represents the delivery status
type MessageStatus string

const (
	MessageStatusSending   MessageStatus = "sending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// MessageFormatType represents the format of message content
type MessageFormatType string

const (
	MessageFormatPlain    MessageFormatType = "plain"
	MessageFormatMarkdown MessageFormatType = "markdown"
	MessageFormatHTML     MessageFormatType = "html"
)

// ChatMessageEvent represents a message event from Kafka
// This struct mirrors the message-service's ChatMessageEvent for deserialization
type ChatMessageEvent struct {
	ID                     uuid.UUID         `json:"ID"`
	ConversationID         uuid.UUID         `json:"ConversationID"`
	SenderUserID           uuid.UUID         `json:"SenderUserID"`
	RecipientIDs           []uuid.UUID       `json:"RecipientIDs,omitempty"`
	ParentMessageID        *uuid.UUID        `json:"ParentMessageID,omitempty"`
	MessageType            MessageType       `json:"MessageType"`
	Content                string            `json:"Content"`
	ContentEncrypted       bool              `json:"ContentEncrypted"`
	ContentHash            string            `json:"ContentHash"`
	FormatType             MessageFormatType `json:"FormatType"`
	Mentions               json.RawMessage   `json:"Mentions,omitempty"`
	Hashtags               []string          `json:"Hashtags,omitempty"`
	Links                  json.RawMessage   `json:"Links,omitempty"`
	Status                 MessageStatus     `json:"Status"`
	DeliveredAt            *time.Time        `json:"DeliveredAt,omitempty"`
	DeliveryCount          int               `json:"DeliveryCount"`
	ReadCount              int               `json:"ReadCount"`
	ReplyCount             int               `json:"ReplyCount"`
	LastReplyAt            *time.Time        `json:"LastReplyAt,omitempty"`
	ReactionCount          int               `json:"ReactionCount"`
	IsForwarded            bool              `json:"IsForwarded"`
	ForwardedFromMessageID *string           `json:"ForwardedFromMessageID,omitempty"`
	ForwardCount           int               `json:"ForwardCount"`
	SentFromDeviceID       string            `json:"SentFromDeviceID"`
	SentFromIP             string            `json:"SentFromIP"`
	CreatedAt              time.Time         `json:"CreatedAt"`
	UpdatedAt              time.Time         `json:"UpdatedAt"`
	Metadata               json.RawMessage   `json:"Metadata,omitempty"`
}

// MessagePayload is the WebSocket payload sent to clients
type MessagePayload struct {
	ID                     uuid.UUID         `json:"id"`
	ConversationID         uuid.UUID         `json:"conversation_id"`
	SenderUserID           uuid.UUID         `json:"sender_user_id"`
	ParentMessageID        *uuid.UUID        `json:"parent_message_id,omitempty"`
	MessageType            MessageType       `json:"message_type"`
	Content                string            `json:"content"`
	ContentEncrypted       bool              `json:"content_encrypted"`
	ContentHash            string            `json:"content_hash,omitempty"`
	FormatType             MessageFormatType `json:"format_type,omitempty"`
	Mentions               json.RawMessage   `json:"mentions,omitempty"`
	Hashtags               []string          `json:"hashtags,omitempty"`
	Links                  json.RawMessage   `json:"links,omitempty"`
	Status                 MessageStatus     `json:"status"`
	DeliveredAt            *time.Time        `json:"delivered_at,omitempty"`
	DeliveryCount          int               `json:"delivery_count"`
	ReadCount              int               `json:"read_count"`
	ReplyCount             int               `json:"reply_count"`
	LastReplyAt            *time.Time        `json:"last_reply_at,omitempty"`
	ReactionCount          int               `json:"reaction_count"`
	IsForwarded            bool              `json:"is_forwarded"`
	ForwardedFromMessageID *string           `json:"forwarded_from_message_id,omitempty"`
	ForwardCount           int               `json:"forward_count"`
	SentFromDeviceID       string            `json:"sent_from_device_id,omitempty"`
	SentFromIP             string            `json:"sent_from_ip,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
	Metadata               json.RawMessage   `json:"metadata,omitempty"`
}

// ToPayload converts ChatMessageEvent to MessagePayload for WebSocket delivery
func (e *ChatMessageEvent) ToPayload() *MessagePayload {
	return &MessagePayload{
		ID:                     e.ID,
		ConversationID:         e.ConversationID,
		SenderUserID:           e.SenderUserID,
		ParentMessageID:        e.ParentMessageID,
		MessageType:            e.MessageType,
		Content:                e.Content,
		ContentEncrypted:       e.ContentEncrypted,
		ContentHash:            e.ContentHash,
		FormatType:             e.FormatType,
		Mentions:               e.Mentions,
		Hashtags:               e.Hashtags,
		Links:                  e.Links,
		Status:                 e.Status,
		DeliveredAt:            e.DeliveredAt,
		DeliveryCount:          e.DeliveryCount,
		ReadCount:              e.ReadCount,
		ReplyCount:             e.ReplyCount,
		LastReplyAt:            e.LastReplyAt,
		ReactionCount:          e.ReactionCount,
		IsForwarded:            e.IsForwarded,
		ForwardedFromMessageID: e.ForwardedFromMessageID,
		ForwardCount:           e.ForwardCount,
		SentFromDeviceID:       e.SentFromDeviceID,
		SentFromIP:             e.SentFromIP,
		CreatedAt:              e.CreatedAt,
		UpdatedAt:              e.UpdatedAt,
		Metadata:               e.Metadata,
	}
}
