package conversation

import "net/http"

type ConversationHandlerInterface interface {
	GetConversations(w http.ResponseWriter, r *http.Request)
	CreateConversation(w http.ResponseWriter, r *http.Request)
}

var _ ConversationHandlerInterface = (*ConversationHandler)(nil)
