package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ClientMessage represents a message received from a client
type ClientMessage struct {
	ID      string            `json:"id"`               // Unique message ID
	Type    ClientMessageType `json:"type"`             // Message type
	Payload json.RawMessage   `json:"payload"`          // Message payload
	Timestamp time.Time       `json:"timestamp,omitempty"` // Client timestamp
}

// ServerMessage represents a message sent to clients
type ServerMessage struct {
	ID        string            `json:"id"`               // Unique message ID
	Type      ServerMessageType `json:"type"`             // Message type
	Payload   interface{}       `json:"payload"`          // Message payload
	Timestamp time.Time         `json:"timestamp"`        // Server timestamp
	RequestID string            `json:"request_id,omitempty"` // Related request ID
}

// ErrorPayload represents error message payload
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// AuthenticatePayload represents authentication payload
type AuthenticatePayload struct {
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
}

// SubscribePayload represents subscription request
type SubscribePayload struct {
	Topics []SubscriptionTopic `json:"topics"`
	Filters map[string]string  `json:"filters,omitempty"` // Topic-specific filters
}

// UnsubscribePayload represents unsubscribe request
type UnsubscribePayload struct {
	Topics []SubscriptionTopic `json:"topics"`
}

// PresenceUpdatePayload represents presence update
type PresenceUpdatePayload struct {
	Status       string `json:"status"`        // online, away, busy, offline
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

// ReadReceiptPayload represents read receipt
type ReadReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
}

// CallSignalingPayload represents WebRTC signaling
type CallSignalingPayload struct {
	CallID       uuid.UUID   `json:"call_id"`
	Participants []uuid.UUID `json:"participants"`
	SDP          string      `json:"sdp,omitempty"`
	ICECandidate string      `json:"ice_candidate,omitempty"`
}

// ConnectedPayload represents successful connection
type ConnectedPayload struct {
	ClientID  string    `json:"client_id"`
	UserID    uuid.UUID `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	ServerTime time.Time `json:"server_time"`
}

// SubscribedPayload represents successful subscription
type SubscribedPayload struct {
	Topics []SubscriptionTopic `json:"topics"`
}

// MessageAckPayload represents message acknowledgment
type MessageAckPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	Status    string    `json:"status"` // received, delivered, read
}
