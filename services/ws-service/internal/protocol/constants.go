package protocol

type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
)

type ErrorCode string

const (
	ErrCodeUnauthorized       ErrorCode = "unauthorized"
	ErrCodeNotImplemented     ErrorCode = "not_implemented"
	ErrCodeInvalidPayload     ErrorCode = "invalid_payload"
	ErrCodeInvalidJSON        ErrorCode = "invalid_json"
	ErrCodeRateLimited        ErrorCode = "rate_limited"
	ErrCodeSubscriptionDenied ErrorCode = "subscription_denied"
	ErrCodeRoutingFailed      ErrorCode = "routing_failed"
	ErrCodeInvalidMessageType ErrorCode = "invalid_message_type"
	ErrCodeInternalError      ErrorCode = "internal_error"
	ErrCodeConnectionClosed   ErrorCode = "connection_closed"
	ErrCodeMessageTooLarge    ErrorCode = "message_too_large"
)

type Topic string

const (
	TopicUser          Topic = "user"
	TopicConversation  Topic = "conversation"
	TopicPresence      Topic = "presence"
	TopicTyping        Topic = "typing"
	TopicCalls         Topic = "calls"
	TopicNotifications Topic = "notifications"
)

type MessageType string

const (
	MsgTypeSubscribe      MessageType = "subscribe"
	MsgTypeUnsubscribe    MessageType = "unsubscribe"
	MsgTypePresenceUpdate MessageType = "presence.update"
	MsgTypePresenceQuery  MessageType = "presence.query"
	MsgTypeTypingStart    MessageType = "typing.start"
	MsgTypeTypingStop     MessageType = "typing.stop"
	MsgTypeMarkRead       MessageType = "mark.read"
	MsgTypeMarkDelivered  MessageType = "mark.delivered"
	MsgTypeCallOffer      MessageType = "call.offer"
	MsgTypeCallAnswer     MessageType = "call.answer"
	MsgTypeCallICE        MessageType = "call.ice"
	MsgTypeCallHangup     MessageType = "call.hangup"
	MsgTypePing           MessageType = "ping"

	MsgTypeSubscribed       MessageType = "subscribed"
	MsgTypeUnsubscribed     MessageType = "unsubscribed"
	MsgTypePresenceResponse MessageType = "presence.response"
	MsgTypeMessageSending   MessageType = "message.sending"
	MsgTypeMessageNew       MessageType = "message.new"
	MsgTypeMessageRead      MessageType = "message.read"
	MsgTypeMessageDelivered MessageType = "message.delivered"
	MsgTypePong             MessageType = "pong"
	MsgTypeError            MessageType = "error"

	MsgTypePresenceChanged  MessageType = "presence.changed"
	MsgTypeConnectionStatus MessageType = "connection.status"
)
