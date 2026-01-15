package model

import (
	"time"

	"github.com/google/uuid"
)

type Mention struct {
	UserID uuid.UUID `json:"user_id"`
	Offset int       `json:"offset"`
	Length int       `json:"length"`
}

type Metadata map[string]interface{}

type SendMessageInput struct {
	ConversationID  uuid.UUID  `json:"conversation_id"`
	SenderUserID    uuid.UUID  `json:"sender_user_id"`
	ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty"`
	Content         string     `json:"content"`
	MessageType     string     `json:"message_type"`
	Mentions        []Mention  `json:"mentions,omitempty"`
	Metadata        Metadata   `json:"metadata,omitempty"`
}

type Message struct {
	ID              uuid.UUID  `json:"id"`
	ConversationID  uuid.UUID  `json:"conversation_id"`
	SenderUserID    uuid.UUID  `json:"sender_user_id"`
	ParentMessageID *uuid.UUID `json:"parent_message_id,omitempty"`
	Content         string     `json:"content"`
	MessageType     string     `json:"message_type"`
	Status          string     `json:"status"`
	IsEdited        bool       `json:"is_edited"`
	IsDeleted       bool       `json:"is_deleted"`
	Mentions        []Mention  `json:"mentions,omitempty"`
	Metadata        Metadata   `json:"metadata,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	EditedAt        *time.Time `json:"edited_at,omitempty"`
	ReadCount       int        `json:"read_count,omitempty"`
}

type PaginationParams struct {
	Limit    int        `json:"limit"`
	BeforeID *uuid.UUID `json:"before_id,omitempty"`
	AfterID  *uuid.UUID `json:"after_id,omitempty"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
}
