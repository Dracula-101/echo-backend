package service

import (
	"context"
	"time"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/messaging"
	"echo-backend/services/message-service/internal/repo"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

const typingExpirationSeconds = 10

// TypingServiceInterface defines the typing indicator service methods
type TypingServiceInterface interface {
	SetTyping(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, isTyping bool) *msgError.MsgError
	GetTypingUsers(ctx context.Context, conversationID uuid.UUID, excludeUserID uuid.UUID) ([]TypingUserResponse, *msgError.MsgError)
}

// TypingUserResponse represents a user currently typing
type TypingUserResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	StartedAt time.Time `json:"started_at"`
}

type typingService struct {
	typingRepo     repo.TypingRepository
	eventPublisher messaging.EventPublisher
	logger         logger.Logger
}

// NewTypingService creates a new typing service
func NewTypingService(typingRepo repo.TypingRepository, eventPublisher messaging.EventPublisher, log logger.Logger) TypingServiceInterface {
	return &typingService{
		typingRepo:     typingRepo,
		eventPublisher: eventPublisher,
		logger:         log,
	}
}

func (s *typingService) SetTyping(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, isTyping bool) *msgError.MsgError {
	if isTyping {
		expiresAt := time.Now().Add(typingExpirationSeconds * time.Second)
		if err := s.typingRepo.UpsertTypingIndicator(ctx, conversationID, userID, expiresAt); err != nil {
			s.logger.Error("Failed to set typing indicator",
				logger.String("conversation_id", conversationID.String()),
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return &msgError.MsgError{
				Message: "Failed to set typing indicator",
				Code:    msgError.CodeConversationUpdateFailed,
				Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to set typing indicator"),
			}
		}
		s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
			EventType:      domain.ConversationEventTypeTypingStarted,
			ConversationID: conversationID,
			UserID:         userID,
			Timestamp:      time.Now(),
		})
	} else {
		if err := s.typingRepo.RemoveTypingIndicator(ctx, conversationID, userID); err != nil {
			s.logger.Error("Failed to remove typing indicator",
				logger.String("conversation_id", conversationID.String()),
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return &msgError.MsgError{
				Message: "Failed to remove typing indicator",
				Code:    msgError.CodeConversationUpdateFailed,
				Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to remove typing indicator"),
			}
		}
		s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
			EventType:      domain.ConversationEventTypeTypingStopped,
			ConversationID: conversationID,
			UserID:         userID,
			Timestamp:      time.Now(),
		})
	}
	return nil
}

func (s *typingService) GetTypingUsers(ctx context.Context, conversationID uuid.UUID, excludeUserID uuid.UUID) ([]TypingUserResponse, *msgError.MsgError) {
	indicators, err := s.typingRepo.GetActiveTypingUsers(ctx, conversationID, excludeUserID)
	if err != nil {
		s.logger.Error("Failed to get typing users",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to get typing users",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get typing users"),
		}
	}

	result := make([]TypingUserResponse, 0, len(indicators))
	for _, ti := range indicators {
		result = append(result, TypingUserResponse{
			UserID:    ti.UserID,
			StartedAt: ti.StartedAt,
		})
	}
	return result, nil
}
