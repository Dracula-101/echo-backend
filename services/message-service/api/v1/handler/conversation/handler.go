package conversation

import (
	svc "echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
)

type ConversationHandler struct {
	conversationService svc.ConversationService
	log                 logger.Logger
}

func NewConversationHandler(conversationService svc.ConversationService, log logger.Logger) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		log:                 log,
	}
}
