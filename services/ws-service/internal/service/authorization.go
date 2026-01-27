package service

import (
	"context"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
	"ws-service/internal/repo"

	"github.com/google/uuid"
)

// AuthorizationService handles permission checks for WebSocket subscriptions
type AuthorizationService interface {
	CanSubscribeToTopic(ctx context.Context, userID uuid.UUID, topic protocol.Topic, resourceID string) (bool, error)
	ValidateConversationAccess(ctx context.Context, userID, conversationID uuid.UUID) (bool, error)
	ValidateUserTopic(ctx context.Context, userID uuid.UUID, targetUserID string) (bool, error)
}

type authorizationService struct {
	conversationRepo repo.ConversationRepository
	userRepo         repo.UserRepository
	log              logger.Logger
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(
	conversationRepo repo.ConversationRepository,
	userRepo repo.UserRepository,
	log logger.Logger,
) AuthorizationService {
	return &authorizationService{
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
		log:              log,
	}
}

// CanSubscribeToTopic validates if a user can subscribe to a specific topic and resource
func (s *authorizationService) CanSubscribeToTopic(ctx context.Context, userID uuid.UUID, topic protocol.Topic, resourceID string) (bool, error) {
	switch topic {
	case protocol.TopicUser:
		return s.ValidateUserTopic(ctx, userID, resourceID)

	case protocol.TopicConversation:
		convID, err := uuid.Parse(resourceID)
		if err != nil {
			s.log.Warn("Invalid conversation ID format",
				logger.String("resource_id", resourceID),
				logger.Error(err),
			)
			return false, nil // Invalid UUID, deny access
		}
		return s.ValidateConversationAccess(ctx, userID, convID)

	case protocol.TopicPresence:
		return true, nil

	case protocol.TopicTyping:
		convID, err := uuid.Parse(resourceID)
		if err != nil {
			s.log.Warn("Invalid conversation ID format for typing topic",
				logger.String("resource_id", resourceID),
				logger.Error(err),
			)
			return false, nil
		}
		return s.ValidateConversationAccess(ctx, userID, convID)

	case protocol.TopicCalls:
		// Calls are per-conversation or per-call, validate conversation access
		// For now, allow authenticated users (can be enhanced later)
		return true, nil

	case protocol.TopicNotifications:
		return s.ValidateUserTopic(ctx, userID, resourceID)

	default:
		s.log.Warn("Unknown topic type",
			logger.String("topic", string(topic)),
			logger.String("user_id", userID.String()),
		)
		return false, nil
	}
}

// ValidateConversationAccess validates if a user can access a conversation
func (s *authorizationService) ValidateConversationAccess(ctx context.Context, userID, conversationID uuid.UUID) (bool, error) {
	allowed, err := s.conversationRepo.ValidateConversationAccess(ctx, userID, conversationID)
	if err != nil {
		s.log.Error("Failed to validate conversation access",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return false, err
	}

	if !allowed {
		s.log.Info("Conversation subscription denied",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
		)
	}

	return allowed, nil
}

// ValidateUserTopic validates if a user can subscribe to a user topic
// Users can only subscribe to their own user topic
func (s *authorizationService) ValidateUserTopic(ctx context.Context, userID uuid.UUID, targetUserID string) (bool, error) {
	// User can only subscribe to their own user-specific topics
	allowed := targetUserID == userID.String()

	if !allowed {
		s.log.Info("User topic subscription denied",
			logger.String("user_id", userID.String()),
			logger.String("target_user_id", targetUserID),
		)
	}

	return allowed, nil
}
