package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Conversation represents a conversation/chat room
type Conversation struct {
	ID               uuid.UUID    `json:"id" db:"id"`
	ConversationType string       `json:"conversation_type" db:"conversation_type"` // direct, group, channel, broadcast
	Title            string       `json:"title,omitempty" db:"title"`
	Description      string       `json:"description,omitempty" db:"description"`
	AvatarURL        *string      `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatorUserID    uuid.UUID    `json:"creator_user_id" db:"creator_user_id"`
	IsEncrypted      bool         `json:"is_encrypted" db:"is_encrypted"`
	IsPublic         bool         `json:"is_public" db:"is_public"`
	IsArchived       bool         `json:"is_archived" db:"is_archived"`
	MemberCount      int          `json:"member_count" db:"member_count"`
	MessageCount     int          `json:"message_count" db:"message_count"`
	LastMessageID    *uuid.UUID   `json:"last_message_id,omitempty" db:"last_message_id"`
	LastMessageAt    *time.Time   `json:"last_message_at,omitempty" db:"last_message_at"`
	LastActivityAt   *time.Time   `json:"last_activity_at,omitempty" db:"last_activity_at"`
	CreatedAt        time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt        sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}
