package conversation

import "shared/server/request"

// ConversationHandlerInterface defines the interface for conversation handlers
type ConversationHandlerInterface interface {
	CreateConversation(handler *request.RequestHandler)
	GetConversationByID(handler *request.RequestHandler)
	GetUserConversations(handler *request.RequestHandler)
}

// Compile-time interface compliance check
var _ ConversationHandlerInterface = (*ConversationHandler)(nil)
