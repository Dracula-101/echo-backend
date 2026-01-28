package service

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

type ConversationService interface {
	CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, *msgError.MessageError)
	GetConversation(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, *msgError.MessageError)
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, *msgError.MessageError)
}

type conversationService struct {
	conversationRepo repo.ConversationRepository
	logger           logger.Logger
}

func NewConversationService(conversationRepo repo.ConversationRepository, log logger.Logger) ConversationService {
	return &conversationService{
		conversationRepo: conversationRepo,
		logger:           log,
	}
}

func (s *conversationService) CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, *msgError.MessageError) {
	conversation, err := s.conversationRepo.CreateConversation(ctx, input)
	if err != nil {
		s.logger.Error("Failed to create conversation",
			logger.String("service", msgError.ServiceName),
			logger.Any("creator_user_id", input.CreatorUserID),
			logger.Int("participant_count", len(input.ParticipantIDs)),
			logger.Bool("is_public", input.IsPublic),
			logger.Error(err),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to create conversation",
			Code:    msgError.CodeConversationCreationFailed,
			Error:   err,
		}
	}
	return conversation, nil
}

func (s *conversationService) GetConversation(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, *msgError.MessageError) {
	exists, err := s.conversationRepo.ConversationExists(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to check conversation existence",
			logger.String("service", msgError.ServiceName),
			logger.Any("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to check conversation existence",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   err,
		}
	}
	if !exists {
		s.logger.Warn("Conversation not found",
			logger.String("service", msgError.ServiceName),
			logger.Any("conversation_id", conversationID),
		)
		return nil, &msgError.MessageError{
			Message: "Conversation not found",
			Code:    msgError.CodeConversationNotFound,
			Error:   msgError.ConversationNotFoundError,
		}
	}

	conversation, err := s.conversationRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to get conversation by ID",
			logger.String("service", msgError.ServiceName),
			logger.Any("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to get conversation",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   err,
		}
	}
	return conversation, nil
}

func (s *conversationService) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, *msgError.MessageError) {
	conversations, err := s.conversationRepo.GetUserConversations(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user conversations",
			logger.String("service", msgError.ServiceName),
			logger.Any("user_id", userID),
			logger.Error(err),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to get user conversations",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   err,
		}
	}
	return conversations, nil
}
