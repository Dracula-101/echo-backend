package conversation

import (
	"echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
)

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	service service.ConversationService
	log     logger.Logger
}

func NewConversationHandler(service service.ConversationService, log logger.Logger) *ConversationHandler {
	return &ConversationHandler{
		service: service,
		log:     log,
	}
}
