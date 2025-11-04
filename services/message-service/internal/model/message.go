package model

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message represents a message in a conversation
type Message struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	ConversationID  uuid.UUID       `json:"conversation_id" db:"conversation_id"`
	SenderUserID    uuid.UUID       `json:"sender_user_id" db:"sender_user_id"`
	ParentMessageID *uuid.UUID      `json:"parent_message_id,omitempty" db:"parent_message_id"`
	Content         string          `json:"content" db:"content"`
	MessageType     string          `json:"message_type" db:"message_type"` // text, image, video, audio, file, location
	Status          string          `json:"status" db:"status"`             // sending, sent, delivered, read, failed
	IsEdited        bool            `json:"is_edited" db:"is_edited"`
	IsDeleted       bool            `json:"is_deleted" db:"is_deleted"`
	Mentions        json.RawMessage `json:"mentions,omitempty" db:"mentions"`
	Metadata        json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt       sql.NullTime    `json:"deleted_at,omitempty" db:"deleted_at"`
	EditedAt        sql.NullTime    `json:"edited_at,omitempty" db:"edited_at"`

	// Joined fields (not in DB)
	SenderName   string `json:"sender_name,omitempty" db:"-"`
	SenderAvatar string `json:"sender_avatar,omitempty" db:"-"`
	ReadCount    int    `json:"read_count,omitempty" db:"-"`
}

// SendMessageRequest represents the request to send a message
type SendMessageRequest struct {
	ConversationID  uuid.UUID  `json:"conversation_id" validate:"required"`
	SenderUserID    uuid.UUID  `json:"sender_user_id" validate:"required"`
	Content         string     `json:"content" validate:"required,max=10000"`
	MessageType     string     `json:"message_type" validate:"required,oneof=text image video audio file location poll"`
	ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty" validate:"omitempty"`
	Mentions        []Mention  `json:"mentions,omitempty"`
	Metadata        Metadata   `json:"metadata,omitempty"`
}

// Mention represents a user mention in a message
type Mention struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	Offset int       `json:"offset" validate:"min=0"`
	Length int       `json:"length" validate:"min=1"`
}

// Metadata contains additional message information
type Metadata struct {
	MediaURL      string  `json:"media_url,omitempty"`
	ThumbnailURL  string  `json:"thumbnail_url,omitempty"`
	FileName      string  `json:"file_name,omitempty"`
	FileSize      int64   `json:"file_size,omitempty"`
	Duration      int     `json:"duration,omitempty"` // for audio/video in seconds
	Width         int     `json:"width,omitempty"`    // for images/videos
	Height        int     `json:"height,omitempty"`   // for images/videos
	Latitude      float64 `json:"latitude,omitempty"` // for location
	Longitude     float64 `json:"longitude,omitempty"`
	LocationName  string  `json:"location_name,omitempty"`
	LinkPreviewID string  `json:"link_preview_id,omitempty"`
}

// DeliveryStatus represents message delivery status for each user
type DeliveryStatus struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	MessageID   uuid.UUID    `json:"message_id" db:"message_id"`
	UserID      uuid.UUID    `json:"user_id" db:"user_id"`
	Status      string       `json:"status" db:"status"` // sent, delivered, read, failed
	DeliveredAt sql.NullTime `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt      sql.NullTime `json:"read_at,omitempty" db:"read_at"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
}

// TypingIndicator represents a user typing in a conversation
type TypingIndicator struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	IsTyping       bool      `json:"is_typing"`
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"` // new_message, message_read, message_delivered, typing, user_online, user_offline
	Payload interface{} `json:"payload"`
}

// MessageEvent represents different message events
type MessageEvent struct {
	Type      string    `json:"type"`
	Message   *Message  `json:"message,omitempty"`
	MessageID uuid.UUID `json:"message_id,omitempty"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadReceipt represents a message read receipt
type ReadReceipt struct {
	MessageID uuid.UUID `json:"message_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	ReadAt    time.Time `json:"read_at"`
}

// ConversationParticipant represents a user in a conversation
type ConversationParticipant struct {
	ID                uuid.UUID    `json:"id" db:"id"`
	ConversationID    uuid.UUID    `json:"conversation_id" db:"conversation_id"`
	UserID            uuid.UUID    `json:"user_id" db:"user_id"`
	Role              string       `json:"role" db:"role"`
	CanSendMessages   bool         `json:"can_send_messages" db:"can_send_messages"`
	LastReadMessageID *uuid.UUID   `json:"last_read_message_id,omitempty" db:"last_read_message_id"`
	LastReadAt        sql.NullTime `json:"last_read_at,omitempty" db:"last_read_at"`
	UnreadCount       int          `json:"unread_count" db:"unread_count"`
	JoinedAt          time.Time    `json:"joined_at" db:"joined_at"`
	LeftAt            sql.NullTime `json:"left_at,omitempty" db:"left_at"`
}

// PaginationParams for message queries
type PaginationParams struct {
	Limit    int        `json:"limit" validate:"min=1,max=100"`
	BeforeID *uuid.UUID `json:"before_id,omitempty"`
	AfterID  *uuid.UUID `json:"after_id,omitempty"`
}

// MessagesResponse represents a paginated messages response
type MessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
	Total    int64     `json:"total,omitempty"`
}
