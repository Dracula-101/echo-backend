package handler

import "net/http"

// MessageHandlerInterface defines the contract for message HTTP handlers
// Note: Message-service primarily uses WebSocket for real-time communication
// HTTP handlers may be added in the future for REST API endpoints
type MessageHandlerInterface interface {
	GetMessages(w http.ResponseWriter, r *http.Request)
}

// Ensure MessageHandler implements MessageHandlerInterface
var _ MessageHandlerInterface = (*MessageHandler)(nil)
