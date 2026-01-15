package message

import (
	"shared/server/request"
)

// MessageHandlerInterface defines the contract for message HTTP handlers
// Note: Message-service primarily uses WebSocket for real-time communication
// HTTP handlers may be added in the future for REST API endpoints
type MessageHandlerInterface interface {
	SendMessage(handler *request.RequestHandler)
}

var _ MessageHandlerInterface = (*MessageHandler)(nil)
