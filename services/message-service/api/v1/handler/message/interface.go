package message

import "net/http"

// MessageHandlerInterface defines the contract for message HTTP handlers
// Note: Message-service primarily uses WebSocket for real-time communication
// HTTP handlers may be added in the future for REST API endpoints
type MessageHandlerInterface interface {
	GetMessages(w http.ResponseWriter, r *http.Request)
	SendMessage(w http.ResponseWriter, r *http.Request)
	EditMessage(w http.ResponseWriter, r *http.Request)
	MarkAsRead(w http.ResponseWriter, r *http.Request)
	SetTypingIndicator(w http.ResponseWriter, r *http.Request)
}

var _ MessageHandlerInterface = (*MessageHandler)(nil)
