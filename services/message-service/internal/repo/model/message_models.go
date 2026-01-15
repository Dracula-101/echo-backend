package model

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message represents a message record stored in the database.
type Message struct {
	ID              uuid.UUID       `db:"id"`
	ConversationID  uuid.UUID       `db:"conversation_id"`
	SenderUserID    uuid.UUID       `db:"sender_user_id"`
	ParentMessageID *uuid.UUID      `db:"parent_message_id"`
	Content         string          `db:"content"`
	ReadCount       int             `db:"read_count"`
	MessageType     string          `db:"message_type"`
	Status          string          `db:"status"`
	IsEdited        bool            `db:"is_edited"`
	IsDeleted       bool            `db:"is_deleted"`
	Mentions        json.RawMessage `db:"mentions"`
	Metadata        json.RawMessage `db:"metadata"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
	DeletedAt       sql.NullTime    `db:"deleted_at"`
	EditedAt        sql.NullTime    `db:"edited_at"`
}

type DeliveryStatus struct {
	ID          uuid.UUID    `db:"id"`
	MessageID   uuid.UUID    `db:"message_id"`
	UserID      uuid.UUID    `db:"user_id"`
	Status      string       `db:"status"`
	DeliveredAt sql.NullTime `db:"delivered_at"`
	ReadAt      sql.NullTime `db:"read_at"`
	CreatedAt   time.Time    `db:"created_at"`
}

type ConversationParticipant struct {
	ID                uuid.UUID    `db:"id"`
	ConversationID    uuid.UUID    `db:"conversation_id"`
	UserID            uuid.UUID    `db:"user_id"`
	Role              string       `db:"role"`
	CanSendMessages   bool         `db:"can_send_messages"`
	LastReadMessageID *uuid.UUID   `db:"last_read_message_id"`
	LastReadAt        sql.NullTime `db:"last_read_at"`
	UnreadCount       int          `db:"unread_count"`
	JoinedAt          time.Time    `db:"joined_at"`
	LeftAt            sql.NullTime `db:"left_at"`
}

type TypingIndicator struct {
	ConversationID uuid.UUID `db:"conversation_id"`
	UserID         uuid.UUID `db:"user_id"`
	IsTyping       bool      `db:"is_typing"`
}

type MessageEvent struct {
	Type      string    `db:"type"`
	MessageID uuid.UUID `db:"message_id"`
	UserID    uuid.UUID `db:"user_id"`
	Status    string    `db:"status"`
	Timestamp time.Time `db:"timestamp"`
}

type PaginationParams struct {
	Limit    int
	BeforeID *uuid.UUID
	AfterID  *uuid.UUID
}
