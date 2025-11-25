package validator

import (
	"encoding/json"
	"fmt"
	"ws-service/internal/ws/protocol"

	"github.com/google/uuid"
)

// MessageValidator validates incoming WebSocket messages
type MessageValidator struct{}

// NewMessageValidator creates a new message validator
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{}
}

// ValidateClientMessage validates a client message
func (v *MessageValidator) ValidateClientMessage(msg *protocol.ClientMessage) error {
	if msg.ID == "" {
		return fmt.Errorf("message ID is required")
	}

	if msg.Type == "" {
		return fmt.Errorf("message type is required")
	}

	// Validate based on message type
	switch msg.Type {
	case protocol.TypeAuthenticate:
		return v.validateAuthenticate(msg.Payload)
	case protocol.TypeSubscribe:
		return v.validateSubscribe(msg.Payload)
	case protocol.TypeUnsubscribe:
		return v.validateUnsubscribe(msg.Payload)
	case protocol.TypePresenceUpdate:
		return v.validatePresenceUpdate(msg.Payload)
	case protocol.TypePresenceQuery:
		return v.validatePresenceQuery(msg.Payload)
	case protocol.TypeTypingStart, protocol.TypeTypingStop:
		return v.validateTyping(msg.Payload)
	case protocol.TypeMarkAsRead, protocol.TypeMarkAsDelivered:
		return v.validateReadReceipt(msg.Payload)
	case protocol.TypeCallOffer, protocol.TypeCallAnswer, protocol.TypeCallICE:
		return v.validateCallSignaling(msg.Payload)
	case protocol.TypePing, protocol.TypeDisconnect, protocol.TypeCallHangup:
		// No payload validation needed
		return nil
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (v *MessageValidator) validateAuthenticate(payload json.RawMessage) error {
	var auth protocol.AuthenticatePayload
	if err := json.Unmarshal(payload, &auth); err != nil {
		return fmt.Errorf("invalid authenticate payload: %w", err)
	}

	if auth.Token == "" {
		return fmt.Errorf("token is required")
	}

	if auth.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}

	return nil
}

func (v *MessageValidator) validateSubscribe(payload json.RawMessage) error {
	var sub protocol.SubscribePayload
	if err := json.Unmarshal(payload, &sub); err != nil {
		return fmt.Errorf("invalid subscribe payload: %w", err)
	}

	if len(sub.Topics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}

	return nil
}

func (v *MessageValidator) validateUnsubscribe(payload json.RawMessage) error {
	var unsub protocol.UnsubscribePayload
	if err := json.Unmarshal(payload, &unsub); err != nil {
		return fmt.Errorf("invalid unsubscribe payload: %w", err)
	}

	if len(unsub.Topics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}

	return nil
}

func (v *MessageValidator) validatePresenceUpdate(payload json.RawMessage) error {
	var presence protocol.PresenceUpdatePayload
	if err := json.Unmarshal(payload, &presence); err != nil {
		return fmt.Errorf("invalid presence update payload: %w", err)
	}

	validStatuses := map[string]bool{
		"online": true, "away": true, "busy": true, "offline": true,
	}

	if !validStatuses[presence.Status] {
		return fmt.Errorf("invalid status: %s", presence.Status)
	}

	return nil
}

func (v *MessageValidator) validatePresenceQuery(payload json.RawMessage) error {
	var query protocol.PresenceQueryPayload
	if err := json.Unmarshal(payload, &query); err != nil {
		return fmt.Errorf("invalid presence query payload: %w", err)
	}

	if len(query.UserIDs) == 0 {
		return fmt.Errorf("at least one user ID is required")
	}

	if len(query.UserIDs) > 100 {
		return fmt.Errorf("maximum 100 user IDs allowed")
	}

	return nil
}

func (v *MessageValidator) validateTyping(payload json.RawMessage) error {
	var typing protocol.TypingPayload
	if err := json.Unmarshal(payload, &typing); err != nil {
		return fmt.Errorf("invalid typing payload: %w", err)
	}

	if typing.ConversationID == uuid.Nil {
		return fmt.Errorf("conversation_id is required")
	}

	return nil
}

func (v *MessageValidator) validateReadReceipt(payload json.RawMessage) error {
	var receipt protocol.ReadReceiptPayload
	if err := json.Unmarshal(payload, &receipt); err != nil {
		return fmt.Errorf("invalid read receipt payload: %w", err)
	}

	if receipt.ConversationID == uuid.Nil {
		return fmt.Errorf("conversation_id is required")
	}

	if len(receipt.MessageIDs) == 0 {
		return fmt.Errorf("at least one message ID is required")
	}

	return nil
}

func (v *MessageValidator) validateCallSignaling(payload json.RawMessage) error {
	var call protocol.CallSignalingPayload
	if err := json.Unmarshal(payload, &call); err != nil {
		return fmt.Errorf("invalid call signaling payload: %w", err)
	}

	if call.CallID == uuid.Nil {
		return fmt.Errorf("call_id is required")
	}

	if len(call.Participants) == 0 {
		return fmt.Errorf("at least one participant is required")
	}

	return nil
}
