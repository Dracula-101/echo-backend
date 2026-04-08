package message

import (
	"shared/server/request"
)

// MessageHandlerInterface defines the contract for message HTTP handlers
// Note: Message-service primarily uses WebSocket for real-time communication
// HTTP handlers may be added in the future for REST API endpoints
type MessageHandlerInterface interface {
	SendMessage(handler *request.RequestHandler)
	GetMessages(handler *request.RequestHandler)
	GetMessageByID(handler *request.RequestHandler)
	EditMessage(handler *request.RequestHandler)
	DeleteMessage(handler *request.RequestHandler)
	AddReaction(handler *request.RequestHandler)
	RemoveReaction(handler *request.RequestHandler)
	GetReactions(handler *request.RequestHandler)
	MarkAsRead(handler *request.RequestHandler)
	GetDeliveryStatus(handler *request.RequestHandler)
	ForwardMessage(handler *request.RequestHandler)
	PinMessage(handler *request.RequestHandler)
	UnpinMessage(handler *request.RequestHandler)
	SearchMessages(handler *request.RequestHandler)
	CreatePoll(handler *request.RequestHandler)
	VotePoll(handler *request.RequestHandler)
	GetPollResults(handler *request.RequestHandler)
}

var _ MessageHandlerInterface = (*MessageHandler)(nil)
