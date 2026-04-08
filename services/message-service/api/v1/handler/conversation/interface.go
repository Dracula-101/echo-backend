package conversation

import "shared/server/request"

// ConversationHandlerInterface defines the interface for conversation handlers
type ConversationHandlerInterface interface {
	CreateConversation(handler *request.RequestHandler)
	GetConversationByID(handler *request.RequestHandler)
	GetUserConversations(handler *request.RequestHandler)
	UpdateConversation(handler *request.RequestHandler)
	DeleteConversation(handler *request.RequestHandler)
	AddParticipants(handler *request.RequestHandler)
	RemoveParticipant(handler *request.RequestHandler)
	UpdateParticipantRole(handler *request.RequestHandler)
	GetPinnedMessages(handler *request.RequestHandler)
	MuteConversation(handler *request.RequestHandler)
	UnmuteConversation(handler *request.RequestHandler)
}

// Compile-time interface compliance check
var _ ConversationHandlerInterface = (*ConversationHandler)(nil)
