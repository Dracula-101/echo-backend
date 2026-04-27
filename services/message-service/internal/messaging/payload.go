package messaging

import (
	"time"

	"github.com/google/uuid"
)

type MessageSentPayload struct {
	MessageID   uuid.UUID `json:"message_id"`
	Body        string    `json:"body"`
	SenderID    uuid.UUID `json:"sender_id"`
	ClientMsgID string    `json:"client_msg_id,omitempty"`
}

type MessageDeletedPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type MessageEditedPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	NewBody   string    `json:"new_body"`
	EditedAt  time.Time `json:"edited_at"`
}

type ReactionPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	Reaction  string    `json:"reaction"`
	ActorID   uuid.UUID `json:"actor_id"`
}

type DeliveryEventType string

const (
	DeliveryEventDelivered DeliveryEventType = "delivered"
	DeliveryEventRead      DeliveryEventType = "read"
)

type DeliveryEvent struct {
	EventType      DeliveryEventType `json:"event_type"`
	MessageIDs     []uuid.UUID       `json:"message_ids"`
	UserID         uuid.UUID         `json:"user_id"`
	ConversationID uuid.UUID         `json:"conversation_id"`
	Timestamp      time.Time         `json:"timestamp"`
}

type ConversationEventType string

const (
	ConversationEventCreated            ConversationEventType = "conversation.created"
	ConversationEventUpdated            ConversationEventType = "conversation.updated"
	ConversationEventParticipantAdded   ConversationEventType = "conversation.participant_added"
	ConversationEventParticipantRemoved ConversationEventType = "conversation.participant_removed"
)

type ConversationEvent struct {
	EventType      ConversationEventType `json:"event_type"`
	ConversationID uuid.UUID             `json:"conversation_id"`
	ActorID        uuid.UUID             `json:"actor_id"`
	Timestamp      time.Time             `json:"timestamp"`
	Payload        any                   `json:"payload"`
}

type ParticipantPayload struct {
	UserID uuid.UUID `json:"user_id"`
}

type ConversationUpdatedPayload struct {
	Title  string `json:"title,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

type ChatMessageEvent struct {
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Body           string    `json:"body"`
	SentAt         time.Time `json:"sent_at"`
}
