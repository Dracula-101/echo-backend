package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const ProtocolVersion = "1.0"

func NewServerMessage(msgType MessageType, requestID string) *ServerMessage {
	return &ServerMessage{
		ID:        uuid.New().String(),
		Type:      string(msgType),
		Version:   ProtocolVersion,
		Status:    StatusSuccess,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
	}
}

func NewSuccessMessage(requestID string, payload any) *ServerMessage {
	return &ServerMessage{
		ID:        uuid.New().String(),
		Type:      string(StatusSuccess),
		Version:   ProtocolVersion,
		Status:    StatusSuccess,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
		Payload:   payload,
	}
}

func NewErrorMessage(requestID string, code ErrorCode, message string) *ServerMessage {
	return &ServerMessage{
		ID:        uuid.New().String(),
		Type:      string(MsgTypeError),
		Version:   ProtocolVersion,
		Status:    StatusError,
		Timestamp: time.Now().UTC(),
		RequestID: requestID,
		Error: &ErrorDetail{
			Code:      code,
			Message:   message,
			Retryable: isRetryableError(code),
		},
	}
}

func NewErrorMessageWithDetails(requestID string, code ErrorCode, message string, details map[string]interface{}) *ServerMessage {
	msg := NewErrorMessage(requestID, code, message)
	msg.Error.Details = details
	return msg
}

func (m *ServerMessage) WithMetadata(serverID string, processingTimeMs int64) *ServerMessage {
	m.Metadata = &MessageMetadata{
		ServerID:         serverID,
		ProcessingTimeMs: processingTimeMs,
	}
	return m
}

func (m *ServerMessage) WithPayload(payload interface{}) *ServerMessage {
	m.Payload = payload
	return m
}

func (m *ServerMessage) WithTraceID(traceID string) *ServerMessage {
	if m.Metadata == nil {
		m.Metadata = &MessageMetadata{}
	}
	m.Metadata.TraceID = traceID
	return m
}

type ClientMessage struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Version        string          `json:"version,omitempty"`
	Timestamp      time.Time       `json:"timestamp,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	ClientInfo     *ClientInfo     `json:"client_info,omitempty"`
}

type ClientInfo struct {
	Platform    string `json:"platform,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	DeviceModel string `json:"device_model,omitempty"`
	OSVersion   string `json:"os_version,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Locale      string `json:"locale,omitempty"`
}

type ServerMessage struct {
	ID        string           `json:"id"`
	Type      string           `json:"type"`
	Version   string           `json:"version"`
	Status    Status           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	RequestID string           `json:"request_id,omitempty"`
	Payload   interface{}      `json:"payload,omitempty"`
	Error     *ErrorDetail     `json:"error,omitempty"`
	Metadata  *MessageMetadata `json:"metadata,omitempty"`
}

type ErrorDetail struct {
	Code           ErrorCode              `json:"code"`
	Message        string                 `json:"message"`
	Details        map[string]interface{} `json:"details,omitempty"`
	Field          string                 `json:"field,omitempty"`
	Retryable      bool                   `json:"retryable"`
	RetryAfterMs   int64                  `json:"retry_after_ms,omitempty"`
	DocumentionURL string                 `json:"documentation_url,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
}

type MessageMetadata struct {
	ServerID         string    `json:"server_id,omitempty"`
	ProcessingTimeMs int64     `json:"processing_time_ms,omitempty"`
	TraceID          string    `json:"trace_id,omitempty"`
	SpanID           string    `json:"span_id,omitempty"`
	Region           string    `json:"region,omitempty"`
	SequenceNumber   int64     `json:"sequence_number,omitempty"`
	AckRequired      bool      `json:"ack_required,omitempty"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
}

type Pagination struct {
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	HasMore    bool   `json:"has_more,omitempty"`
	TotalCount int    `json:"total_count,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

type ResourceRef struct {
	Type string    `json:"type"`
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name,omitempty"`
}

type UserRef struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
}

type ConversationRef struct {
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type,omitempty"`
	Title string    `json:"title,omitempty"`
}

type MessageRef struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id,omitempty"`
	Preview        string    `json:"preview,omitempty"`
}

type Acknowledgment struct {
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
	Status    AckStatus `json:"status"`
}

type AckStatus string

const (
	AckStatusReceived  AckStatus = "received"
	AckStatusProcessed AckStatus = "processed"
	AckStatusFailed    AckStatus = "failed"
)
