package protocol

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

func isValidTopic(topic Topic) bool {
	switch topic {
	case TopicUser,
		TopicConversation,
		TopicPresence,
		TopicTyping,
		TopicCalls,
		TopicNotifications:
		return true
	default:
		return false
	}
}

func containsTopic(topics []Topic, targets ...Topic) bool {
	for _, topic := range topics {
		if slices.Contains(targets, topic) {
			return true
		}
	}
	return false
}

type SubscribePayload struct {
	Topics  []Topic        `json:"topics"`
	Filters MessageFilters `json:"filters"`
}

func ValidateSubscribePayload(payload SubscribePayload) bool {
	if len(payload.Topics) == 0 {
		return false
	}
	for _, topic := range payload.Topics {
		if !isValidTopic(topic) {
			return false
		}
	}
	if containsTopic(payload.Topics, TopicConversation) {
		if payload.Filters.ConversationIDs == nil || len(*payload.Filters.ConversationIDs) == 0 {
			return false
		}
	}
	if containsTopic(payload.Topics, TopicUser, TopicPresence, TopicTyping, TopicNotifications, TopicCalls) {
		if payload.Filters.UserID == nil {
			return false
		}
	}
	return true
}

type UnsubscribePayload struct {
	Topics  []Topic        `json:"topics"`
	Filters MessageFilters `json:"filters,omitempty"`
}

func ValidateUnsubscribePayload(payload UnsubscribePayload) bool {
	if len(payload.Topics) == 0 {
		return false
	}
	for _, topic := range payload.Topics {
		if !isValidTopic(topic) {
			return false
		}
	}
	if containsTopic(payload.Topics, TopicConversation) {
		if payload.Filters.ConversationIDs == nil || len(*payload.Filters.ConversationIDs) == 0 {
			return false
		}
	}
	return true
}

type PresenceUpdatePayload struct {
	Status       string                 `json:"status"`
	CustomStatus string                 `json:"custom_status,omitempty"`
	DeviceInfo   map[string]interface{} `json:"device_info,omitempty"`
}

func ValidatePresenceUpdatePayload(payload PresenceUpdatePayload) bool {
	allowedStatuses := map[string]bool{
		"online":  true,
		"offline": true,
		"away":    true,
		"busy":    true,
	}
	return allowedStatuses[payload.Status]
}

type PresenceQueryPayload struct {
	UserIDs []uuid.UUID `json:"user_ids"`
}

func ValidatePresenceQueryPayload(payload PresenceQueryPayload) bool {
	return len(payload.UserIDs) > 0
}

type TypingPayload struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}

func ValidateTypingPayload(payload TypingPayload) bool {
	return payload.ConversationID != uuid.Nil
}

type ReadReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	ReadAt         time.Time   `json:"read_at,omitempty"`
}

func ValidateReadReceiptPayload(payload ReadReceiptPayload) bool {
	return payload.ConversationID != uuid.Nil && len(payload.MessageIDs) > 0
}

type DeliveredReceiptPayload struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
	DeliveredAt    time.Time   `json:"delivered_at,omitempty"`
}

func ValidateDeliveredReceiptPayload(payload DeliveredReceiptPayload) bool {
	return payload.ConversationID != uuid.Nil && len(payload.MessageIDs) > 0
}

type PingPayload struct {
	ClientTimestamp time.Time         `json:"client_timestamp,omitempty"`
	NetworkInfo     map[string]string `json:"network_info,omitempty"`
}

func ValidatePingPayload(payload PingPayload) bool {
	return true
}
