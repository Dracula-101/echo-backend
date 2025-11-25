package protocol

// ClientMessageType represents message types sent by clients
type ClientMessageType string

const (
	// Connection lifecycle
	TypePing          ClientMessageType = "ping"
	TypePongResponse  ClientMessageType = "pong"
	TypeAuthenticate  ClientMessageType = "authenticate"
	TypeDisconnect    ClientMessageType = "disconnect"

	// Subscriptions
	TypeSubscribe   ClientMessageType = "subscribe"
	TypeUnsubscribe ClientMessageType = "unsubscribe"

	// Presence
	TypePresenceUpdate ClientMessageType = "presence.update"
	TypePresenceQuery  ClientMessageType = "presence.query"

	// Typing indicators
	TypeTypingStart ClientMessageType = "typing.start"
	TypeTypingStop  ClientMessageType = "typing.stop"

	// Read receipts
	TypeMarkAsRead     ClientMessageType = "mark.read"
	TypeMarkAsDelivered ClientMessageType = "mark.delivered"

	// Call signaling
	TypeCallOffer    ClientMessageType = "call.offer"
	TypeCallAnswer   ClientMessageType = "call.answer"
	TypeCallICE      ClientMessageType = "call.ice"
	TypeCallHangup   ClientMessageType = "call.hangup"

	// Message operations
	TypeMessageAck ClientMessageType = "message.ack"
)

// ServerMessageType represents message types sent by server
type ServerMessageType string

const (
	// Connection lifecycle
	ServerTypePing             ServerMessageType = "ping"
	ServerTypePong             ServerMessageType = "pong"
	ServerTypeConnected        ServerMessageType = "connected"
	ServerTypeAuthSuccess      ServerMessageType = "auth.success"
	ServerTypeAuthFailed       ServerMessageType = "auth.failed"
	ServerTypeError            ServerMessageType = "error"
	ServerTypeDisconnected     ServerMessageType = "disconnected"

	// Subscriptions
	ServerTypeSubscribed   ServerMessageType = "subscribed"
	ServerTypeUnsubscribed ServerMessageType = "unsubscribed"

	// Presence
	ServerTypePresenceUpdate  ServerMessageType = "presence.update"
	ServerTypePresenceOnline  ServerMessageType = "presence.online"
	ServerTypePresenceOffline ServerMessageType = "presence.offline"

	// Typing indicators
	ServerTypeTypingStart ServerMessageType = "typing.start"
	ServerTypeTypingStop  ServerMessageType = "typing.stop"

	// Messages
	ServerTypeMessageNew       ServerMessageType = "message.new"
	ServerTypeMessageDelivered ServerMessageType = "message.delivered"
	ServerTypeMessageRead      ServerMessageType = "message.read"
	ServerTypeMessageEdited    ServerMessageType = "message.edited"
	ServerTypeMessageDeleted   ServerMessageType = "message.deleted"

	// Call signaling
	ServerTypeCallIncoming ServerMessageType = "call.incoming"
	ServerTypeCallOffer    ServerMessageType = "call.offer"
	ServerTypeCallAnswer   ServerMessageType = "call.answer"
	ServerTypeCallICE      ServerMessageType = "call.ice"
	ServerTypeCallEnded    ServerMessageType = "call.ended"

	// Notifications
	ServerTypeNotification ServerMessageType = "notification"
)

// SubscriptionTopic represents topics clients can subscribe to
type SubscriptionTopic string

const (
	TopicUser         SubscriptionTopic = "user"         // User-specific events
	TopicConversation SubscriptionTopic = "conversation" // Conversation events
	TopicPresence     SubscriptionTopic = "presence"     // Presence updates
	TopicTyping       SubscriptionTopic = "typing"       // Typing indicators
	TopicCalls        SubscriptionTopic = "calls"        // Call events
	TopicNotifications SubscriptionTopic = "notifications" // Notifications
)
