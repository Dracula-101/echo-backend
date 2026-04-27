package service

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/messaging"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

type ConversationService interface {
	CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, *msgError.MsgError)
	GetConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (*domain.Conversation, *msgError.MsgError)
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, *msgError.MsgError)
	UpdateConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, input domain.UpdateConversationInput) (*domain.Conversation, *msgError.MsgError)
	DeleteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) *msgError.MsgError
	AddParticipants(ctx context.Context, input domain.AddParticipantsInput) *msgError.MsgError
	RemoveParticipant(ctx context.Context, input domain.RemoveParticipantInput) *msgError.MsgError
	UpdateParticipantRole(ctx context.Context, input domain.UpdateParticipantRoleInput) *msgError.MsgError
	MuteConversation(ctx context.Context, input domain.MuteConversationInput) *msgError.MsgError
	UnmuteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) *msgError.MsgError
	GetUnreadCount(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (int, *msgError.MsgError)
}

type conversationService struct {
	conversationRepo repo.ConversationRepository
	userRepo         repo.UserRepository
	eventPublisher   messaging.EventPublisher
	cache            cache.MessageCache
	logger           logger.Logger
}

func NewConversationService(conversationRepo repo.ConversationRepository, userRepo repo.UserRepository, eventPublisher messaging.EventPublisher, cache cache.MessageCache, log logger.Logger) ConversationService {
	return &conversationService{
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
		eventPublisher:   eventPublisher,
		cache:            cache,
		logger:           log,
	}
}

func (s *conversationService) CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, *msgError.MsgError) {
	conversation, err := s.conversationRepo.CreateConversation(ctx, input)
	if err != nil {
		s.logger.Error("Failed to create conversation",
			logger.String("service", msgError.ServiceName),
			logger.Any("creator_user_id", input.CreatorUserID),
			logger.Int("participant_count", len(input.ParticipantIDs)),
			logger.Bool("is_public", input.IsPublic),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to create conversation",
			Code:    msgError.CodeConversationCreationFailed,
			Error:   err,
		}
	}
	return conversation, nil
}

func (s *conversationService) GetConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (*domain.Conversation, *msgError.MsgError) {
	exists, err := s.conversationRepo.ConversationExists(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to check conversation existence",
			logger.String("service", msgError.ServiceName),
			logger.Any("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
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
		return nil, &msgError.MsgError{
			Message: "Conversation not found",
			Code:    msgError.CodeConversationNotFound,
			Error:   pkgErrors.New(msgError.CodeConversationNotFound.String(), "conversation does not exist"),
		}
	}

	conversation, err := s.conversationRepo.GetConversationByID(ctx, conversationID, &userID)
	if err != nil {
		s.logger.Error("Failed to get conversation by ID",
			logger.String("service", msgError.ServiceName),
			logger.Any("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to get conversation",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   err,
		}
	}
	return conversation, nil
}

func (s *conversationService) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, *msgError.MsgError) {
	conversations, err := s.conversationRepo.GetUserConversations(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user conversations",
			logger.String("service", msgError.ServiceName),
			logger.Any("user_id", userID),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to get user conversations",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   err,
		}
	}
	return conversations, nil
}
