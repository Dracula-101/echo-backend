package handler

// MessageHandlerInterface defines the contract for message HTTP handlers
// Note: Message-service primarily uses WebSocket for real-time communication
// HTTP handlers may be added in the future for REST API endpoints
type MessageHandlerInterface interface {
	// Future HTTP handler methods will be defined here
	// Current implementation uses WebSocket hub instead
}

// Ensure MessageHandler implements MessageHandlerInterface
var _ MessageHandlerInterface = (*MessageHandler)(nil)
