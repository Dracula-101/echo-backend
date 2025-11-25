package event

// Common event types
const (
	// Connection events
	EventConnecting    = "connecting"
	EventConnected     = "connected"
	EventDisconnecting = "disconnecting"
	EventDisconnected  = "disconnected"
	EventReconnecting  = "reconnecting"
	EventError         = "error"

	// Message events
	EventMessageReceived = "message.received"
	EventMessageSent     = "message.sent"
	EventMessageError    = "message.error"

	// State events
	EventStateChange = "state.change"

	// Hub events
	EventClientRegistered   = "client.registered"
	EventClientUnregistered = "client.unregistered"
)
