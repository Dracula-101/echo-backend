package domain

import (
	"time"

	"github.com/google/uuid"
)

// MessageEventType represents the type of message event
type MessageEventType string

const (
	MessageEventTypeSent      MessageEventType = "message.sent"
	MessageEventTypeEdited    MessageEventType = "message.edited"
	MessageEventTypeDeleted   MessageEventType = "message.deleted"
	MessageEventTypeDelivered MessageEventType = "message.delivered"
	MessageEventTypeRead      MessageEventType = "message.read"
	MessageEventTypeFailed    MessageEventType = "message.failed"
)

// ConversationEvent represents a conversation-related event
type ConversationEvent struct {
	EventType      ConversationEventType
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Timestamp      time.Time
	Payload        interface{}
}

// ConversationEventType represents the type of conversation event
type ConversationEventType string

const (
	ConversationEventTypeCreated          ConversationEventType = "conversation.created"
	ConversationEventTypeUpdated          ConversationEventType = "conversation.updated"
	ConversationEventTypeDeleted          ConversationEventType = "conversation.deleted"
	ConversationEventTypeParticipantAdded ConversationEventType = "conversation.participant_added"
	ConversationEventTypeParticipantLeft  ConversationEventType = "conversation.participant_left"
	ConversationEventTypeTypingStarted    ConversationEventType = "conversation.typing_started"
	ConversationEventTypeTypingStopped    ConversationEventType = "conversation.typing_stopped"
)
