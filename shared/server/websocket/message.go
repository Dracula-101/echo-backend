package websocket

import (
	"encoding/json"
	"time"
)

type OutgoingMessage struct {
	ID        string      `json:"id,omitempty"`
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

func NewOutgoingMessage(msgType string, payload interface{}) *OutgoingMessage {
	return &OutgoingMessage{
		ID:        generateMessageID(),
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

func NewOutgoingMessageWithID(id, msgType string, payload interface{}) *OutgoingMessage {
	return &OutgoingMessage{
		ID:        id,
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

func (m *OutgoingMessage) WithRequestID(requestID string) *OutgoingMessage {
	m.RequestID = requestID
	return m
}

func (m *OutgoingMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *OutgoingMessage) ToBytes() ([]byte, error) {
	return m.ToJSON()
}

type IncomingMessage struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
}

func ParseIncomingMessage(data []byte) (*IncomingMessage, error) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, NewMessageError("", "unknown", err, "failed to parse message")
	}
	return &msg, nil
}

func (m *IncomingMessage) UnmarshalPayload(v interface{}) error {
	if err := json.Unmarshal(m.Payload, v); err != nil {
		return NewMessageError("", m.Type, err, "failed to unmarshal payload")
	}
	return nil
}

type ErrorMessage struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

func NewErrorMessage(code, message string, details interface{}) *OutgoingMessage {
	return NewOutgoingMessage("error", &ErrorMessage{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	})
}

func NewErrorMessageWithRequestID(code, message, requestID string, details interface{}) *OutgoingMessage {
	return NewOutgoingMessage("error", &ErrorMessage{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		RequestID: requestID,
	})
}

type SuccessMessage struct {
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

func NewSuccessMessage(message string, data interface{}) *OutgoingMessage {
	return NewOutgoingMessage("success", &SuccessMessage{
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

func NewSuccessMessageWithRequestID(message, requestID string, data interface{}) *OutgoingMessage {
	return NewOutgoingMessage("success", &SuccessMessage{
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: requestID,
	})
}

func generateMessageID() string {
	return "msg_" + generateClientID()
}

type MessageBuilder struct {
	msgType   string
	payload   map[string]interface{}
	requestID string
	id        string
}

func NewMessageBuilder(msgType string) *MessageBuilder {
	return &MessageBuilder{
		msgType: msgType,
		payload: make(map[string]interface{}),
	}
}

func (mb *MessageBuilder) WithID(id string) *MessageBuilder {
	mb.id = id
	return mb
}

func (mb *MessageBuilder) WithRequestID(requestID string) *MessageBuilder {
	mb.requestID = requestID
	return mb
}

func (mb *MessageBuilder) WithField(key string, value interface{}) *MessageBuilder {
	mb.payload[key] = value
	return mb
}

func (mb *MessageBuilder) WithFields(fields map[string]interface{}) *MessageBuilder {
	for k, v := range fields {
		mb.payload[k] = v
	}
	return mb
}

func (mb *MessageBuilder) Build() *OutgoingMessage {
	msg := &OutgoingMessage{
		Type:      mb.msgType,
		Payload:   mb.payload,
		Timestamp: time.Now(),
		RequestID: mb.requestID,
	}

	if mb.id != "" {
		msg.ID = mb.id
	} else {
		msg.ID = generateMessageID()
	}

	return msg
}

const (
	MsgTypeConnected    = "connected"
	MsgTypeDisconnected = "disconnected"
	MsgTypeError        = "error"
	MsgTypeSuccess      = "success"
	MsgTypePing         = "ping"
	MsgTypePong         = "pong"
)

func ConnectedMessage(clientID string, additionalData map[string]interface{}) *OutgoingMessage {
	builder := NewMessageBuilder(MsgTypeConnected).
		WithField("client_id", clientID).
		WithField("connected_at", time.Now())

	if additionalData != nil {
		builder.WithFields(additionalData)
	}

	return builder.Build()
}

func DisconnectedMessage(reason string) *OutgoingMessage {
	return NewMessageBuilder(MsgTypeDisconnected).
		WithField("reason", reason).
		WithField("disconnected_at", time.Now()).
		Build()
}

func ErrorMessageSimple(code, message string) *OutgoingMessage {
	return NewErrorMessage(code, message, nil)
}

func SuccessMessageSimple(message string) *OutgoingMessage {
	return NewSuccessMessage(message, nil)
}
