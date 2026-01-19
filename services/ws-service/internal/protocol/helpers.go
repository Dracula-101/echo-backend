package protocol

import (
	"time"

	"github.com/google/uuid"
)

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

func isRetryableError(code ErrorCode) bool {
	switch code {
	case ErrCodeRateLimited, ErrCodeInternalError:
		return true
	default:
		return false
	}
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
