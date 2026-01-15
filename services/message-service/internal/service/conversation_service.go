package service

import (
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/logger"
)

type conversationService struct {
	conversationRepo repo.ConversationRepository
	logger           logger.Logger
}

func NewConversationService(conversationRepo repo.ConversationRepository, log logger.Logger) ConversationServiceInterface {
	return &conversationService{
		conversationRepo: conversationRepo,
		logger:           log,
	}
}
