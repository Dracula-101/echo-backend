package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ClientMessage represents a message from client
type ClientMessage struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
}

// ServerMessage represents a message to client
type ServerMessage struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorPayload represents error response
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Topic represents subscription topics
type Topic string

const (
	TopicUser          Topic = "user"
	TopicConversation  Topic = "conversation"
	TopicPresence      Topic = "presence"
	TopicTyping        Topic = "typing"
	TopicCalls         Topic = "calls"
	TopicNotifications Topic = "notifications"
)

// SubscribePayload represents subscription request
type SubscribePayload struct {
	Topics  []Topic           `json:"topics"`
	Filters map[string]string `json:"filters,omitempty"`
}

// UnsubscribePayload represents unsubscribe request
type UnsubscribePayload struct {
	Topics []Topic `json:"topics"`
}

// SubscribedPayload represents subscription confirmation
type SubscribedPayload struct {
	Topics []Topic `json:"topics"`
}

// UnsubscribedPayload represents unsubscription confirmation
type UnsubscribedPayload struct {
	Topics []Topic `json:"topics"`
}

// PresenceUpdatePayload represents presence update
type PresenceUpdatePayload struct {
	Status       string `json:"status"`
	CustomStatus string `json:"custom_status,omitempty"`
}

// PresenceQueryPayload represents presence query
type PresenceQueryPayload struct {
	UserIDs []uuid.UUID `json:"user_ids"`
}

// TypingPayload represents typing indicator
type TypingPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool      `json:"is_typing"`
}

// TypingEvent represents typing event
type TypingEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool      `json:"is_typing"`
	Timestamp      time.Time `json:"timestamp"`
}

// ReadReceiptPayload represents read receipt
type ReadReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
}

// DeliveredReceiptPayload represents delivery receipt
type DeliveredReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
}

// ReadReceiptEvent represents read receipt event
type ReadReceiptEvent struct {
	UserID         uuid.UUID   `json:"user_id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	Timestamp      time.Time   `json:"timestamp"`
}

// DeliveredReceiptEvent represents delivery receipt event
type DeliveredReceiptEvent struct {
	UserID         uuid.UUID   `json:"user_id"`
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	Timestamp      time.Time   `json:"timestamp"`
}

// GetResourceID extracts resource ID from filters based on topic
func GetResourceID(topic Topic, filters map[string]string) string {
	switch topic {
	case TopicUser:
		if userID, ok := filters["user_id"]; ok {
			return userID
		}
	case TopicConversation:
		if convID, ok := filters["conversation_id"]; ok {
			return convID
		}
	case TopicPresence:
		return "global"
	case TopicTyping:
		if convID, ok := filters["conversation_id"]; ok {
			return convID
		}
	case TopicCalls:
		if callID, ok := filters["call_id"]; ok {
			return callID
		}
	case TopicNotifications:
		if userID, ok := filters["user_id"]; ok {
			return userID
		}
	}
	return "default"
}
