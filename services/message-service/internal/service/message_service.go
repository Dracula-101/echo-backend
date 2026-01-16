package service

import (
	"context"

	"echo-backend/services/message-service/internal/domain"
	"echo-backend/services/message-service/internal/repo"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type messageService struct {
	messageRepo      repo.MessageRepository
	conversationRepo repo.ConversationRepository
	eventPublisher   EventPublisher
	logger           logger.Logger
}

func NewMessageService(
	messageRepo repo.MessageRepository,
	conversationRepo repo.ConversationRepository,
	eventPublisher EventPublisher,
	log logger.Logger,
) MessageServiceInterface {
	return &messageService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		eventPublisher:   eventPublisher,
		logger:           log,
	}
}

// SendMessage handles the complete flow of sending a message
func (s *messageService) SendMessage(ctx context.Context, req *domain.SendMessageInput) (*domain.Message, pkgErrors.AppError) {

	return nil, nil
}
