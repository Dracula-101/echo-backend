package protocol

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
	MsgTypeSyncRequired     MessageType = "sync.required"
)
