package service

import (
	"context"
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

type ConversationService interface {
	CreateConversation(userID uuid.UUID, conversationType string, participantIDs []uuid.UUID, title, description string, isEncrypted, isPublic bool) (uuid.UUID, []uuid.UUID, int64, error)
	GetConversations(userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int, error)
}

type conversationService struct {
	repo   repo.ConversationRepository
	logger logger.Logger
}

func NewConversationService(repo repo.ConversationRepository, log logger.Logger) ConversationService {
	return &conversationService{
		repo:   repo,
		logger: log,
	}
}

// CreateConversation creates a new conversation with participants
func (s *conversationService) CreateConversation(
	userID uuid.UUID,
	conversationType string,
	participantIDs []uuid.UUID,
	title, description string,
	isEncrypted, isPublic bool,
) (uuid.UUID, []uuid.UUID, int64, error) {
	ctx := context.Background()

	s.logger.Info("Creating conversation",
		logger.String("user_id", userID.String()),
		logger.String("conversation_type", conversationType),
		logger.Int("participant_count", len(participantIDs)),
	)

	// Create the conversation
	conversationID, err := s.repo.CreateConversation(
		ctx,
		conversationType,
		title,
		description,
		userID,
		isEncrypted,
		isPublic,
	)
	if err != nil {
		s.logger.Error("Failed to create conversation",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return uuid.Nil, nil, 0, err
	}

	// Add creator as first participant with owner role
	err = s.repo.AddParticipants(ctx, conversationID, []uuid.UUID{userID}, "owner", true)
	if err != nil {
		s.logger.Error("Failed to add creator as participant",
			logger.String("conversation_id", conversationID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return uuid.Nil, nil, 0, err
	}

	// Add other participants
	if len(participantIDs) > 0 {
		// Filter out creator if they're in the participant list
		otherParticipants := make([]uuid.UUID, 0, len(participantIDs))
		for _, pid := range participantIDs {
			if pid != userID {
				otherParticipants = append(otherParticipants, pid)
			}
		}

		if len(otherParticipants) > 0 {
			err = s.repo.AddParticipants(ctx, conversationID, otherParticipants, "member", true)
			if err != nil {
				s.logger.Error("Failed to add participants",
					logger.String("conversation_id", conversationID.String()),
					logger.Error(err),
				)
				return uuid.Nil, nil, 0, err
			}
		}
	}

	// Build complete participant list (creator + others)
	allParticipants := []uuid.UUID{userID}
	for _, pid := range participantIDs {
		if pid != userID {
			allParticipants = append(allParticipants, pid)
		}
	}

	s.logger.Info("Conversation created successfully",
		logger.String("conversation_id", conversationID.String()),
		logger.String("user_id", userID.String()),
		logger.Int("total_participants", len(allParticipants)),
	)

	// Return conversation ID, all participants, and creation timestamp
	return conversationID, allParticipants, 0, nil // TODO: return actual created_at timestamp
}

// GetConversations retrieves conversations for a user
func (s *conversationService) GetConversations(userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int, error) {
	ctx := context.Background()

	s.logger.Debug("Getting conversations",
		logger.String("user_id", userID.String()),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	conversations, total, err := s.repo.GetConversationsByUserID(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, 0, err
	}

	s.logger.Debug("Conversations retrieved",
		logger.String("user_id", userID.String()),
		logger.Int("count", len(conversations)),
		logger.Int("total", total),
	)

	return conversations, total, nil
}
