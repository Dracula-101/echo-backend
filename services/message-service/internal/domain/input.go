package domain

import (
	"time"

	"github.com/google/uuid"
)

// SendMessageInput contains data for sending a message
type SendMessageInput struct {
	ConversationID  uuid.UUID
	SenderUserID    uuid.UUID
	Content         string
	MessageType     MessageType
	ParentMessageID *uuid.UUID
	Metadata        *MessageMetadata
}

// EditMessageInput contains data for editing a message
type EditMessageInput struct {
	MessageID  uuid.UUID
	UserID     uuid.UUID
	NewContent string
}

// DeleteMessageInput contains data for deleting a message
type DeleteMessageInput struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
}

// MarkAsReadInput contains data for marking messages as read
type MarkAsReadInput struct {
	MessageIDs []uuid.UUID
	UserID     uuid.UUID
}

// GetMessagesInput contains parameters for fetching messages
type GetMessagesInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Limit          int
	BeforeID       *uuid.UUID // Cursor: messages before this ID
	AfterID        *uuid.UUID // Cursor: messages after this ID
	BeforeTime     *time.Time // Time-based cursor
	AfterTime      *time.Time // Time-based cursor
}

// CreateConversationInput contains data for creating a conversation
type CreateConversationInput struct {
	CreatorUserID    uuid.UUID
	ConversationType ConversationType
	Title            string
	Description      string
	AvatarURL        *string
	ParticipantIDs   []uuid.UUID
	IsEncrypted      bool
	IsPublic         bool
	MemberCount      int
}

// GetConversationsInput contains parameters for fetching conversations
type GetConversationsInput struct {
	UserID           uuid.UUID
	Limit            int
	Offset           int
	IncludeArchived  bool
	ConversationType *ConversationType
}

// AddParticipantsInput contains data for adding participants
type AddParticipantsInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID // User adding participants
	ParticipantIDs []uuid.UUID
}

// RemoveParticipantInput contains data for removing a participant
type RemoveParticipantInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID // User removing participant
	ParticipantID  uuid.UUID
}

// SetTypingInput contains data for setting typing indicator
type SetTypingInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	IsTyping       bool // true = started typing, false = stopped typing
}

// GetTypingUsersInput contains parameters for fetching typing users
type GetTypingUsersInput struct {
	ConversationID uuid.UUID
	ExcludeUserID  uuid.UUID // Don't include this user (the requester)
}

// MarkAsDeliveredInput contains data for marking message as delivered
type MarkAsDeliveredInput struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
}

// GetDeliveryStatusInput contains parameters for fetching delivery status
type GetDeliveryStatusInput struct {
	MessageID uuid.UUID
	UserID    *uuid.UUID // If nil, get all statuses for message
}
