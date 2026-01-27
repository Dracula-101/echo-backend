package protocol

import (
	"time"

	"github.com/google/uuid"
)

type SubscribePayload struct {
	Topics          []Topic           `json:"topics"`
	Filters         map[string]string `json:"filters,omitempty"`
	ConversationIDs []uuid.UUID       `json:"conversation_ids,omitempty"` // Batch subscribe to multiple conversations
}

type UnsubscribePayload struct {
	Topics          []Topic           `json:"topics"`
	Filters         map[string]string `json:"filters,omitempty"`
	ConversationIDs []uuid.UUID       `json:"conversation_ids,omitempty"` // Batch unsubscribe from multiple conversations
}

type PresenceUpdatePayload struct {
	Status       string                 `json:"status"`
	CustomStatus string                 `json:"custom_status,omitempty"`
	DeviceInfo   map[string]interface{} `json:"device_info,omitempty"`
}

type PresenceQueryPayload struct {
	UserIDs []uuid.UUID `json:"user_ids"`
}

type TypingPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}

type ReadReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	ReadAt         time.Time   `json:"read_at,omitempty"`
}

type DeliveredReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	DeliveredAt    time.Time   `json:"delivered_at,omitempty"`
}

type PingPayload struct {
	ClientTimestamp time.Time         `json:"client_timestamp,omitempty"`
	NetworkInfo     map[string]string `json:"network_info,omitempty"`
}
