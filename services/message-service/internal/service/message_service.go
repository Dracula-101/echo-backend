package service

import (
	"context"

	"echo-backend/services/message-service/internal/repo"
	svcmodel "echo-backend/services/message-service/internal/service/model"

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
func (s *messageService) SendMessage(ctx context.Context, req *svcmodel.SendMessageInput) (*svcmodel.Message, pkgErrors.AppError) {
	
	return nil, nil
}
