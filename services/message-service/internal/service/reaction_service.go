package service

import (
	"context"
	"time"

	"echo-backend/services/message-service/internal/domain"
	"echo-backend/services/message-service/internal/messaging"
	"echo-backend/services/message-service/internal/repo"
	dbModel "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

type ReactionSummaryEntry struct {
	ReactionType string `json:"reaction_type"`
	Count        int    `json:"count"`
	ReactedByMe  bool   `json:"reacted_by_me"`
}

type ReactionServiceInterface interface {
	AddReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reactionType string, emoji *string, skinTone *string) pkgErrors.AppError
	RemoveReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reactionType string) pkgErrors.AppError
	GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]dbModel.Reaction, pkgErrors.AppError)
	GetMessageReactionsWithSummary(ctx context.Context, messageID uuid.UUID, requesterUserID uuid.UUID) ([]dbModel.Reaction, []ReactionSummaryEntry, pkgErrors.AppError)
}

type reactionService struct {
	reactionRepo     repo.ReactionRepository
	messageRepo      repo.MessageRepository
	conversationRepo repo.ConversationRepository
	eventPublisher   messaging.EventPublisher
	logger           logger.Logger
}

func NewReactionService(
	reactionRepo repo.ReactionRepository,
	messageRepo repo.MessageRepository,
	conversationRepo repo.ConversationRepository,
	eventPublisher messaging.EventPublisher,
	log logger.Logger,
) ReactionServiceInterface {
	return &reactionService{
		reactionRepo:     reactionRepo,
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		eventPublisher:   eventPublisher,
		logger:           log,
	}
}

func (s *reactionService) AddReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reactionType string, emoji *string, skinTone *string) pkgErrors.AppError {
	// 1. Verify message exists
	msg, err := s.messageRepo.GetMessageByID(ctx, messageID.String())
	if err != nil {
		s.logger.Error("Message not found for reaction", logger.String("message_id", messageID.String()))
		return pkgErrors.FromError(err, pkgErrors.CodeNotFound, "Message not found")
	}

	// 2. Verify user is participant of the conversation
	if err := s.conversationRepo.ValidateConversationParticipant(uuid.MustParse(msg.ConversationID), userID); err != nil {
		s.logger.Warn("User not in conversation, cannot react", logger.String("user_id", userID.String()))
		return pkgErrors.FromError(err, pkgErrors.CodePermissionDenied, "Cannot react to message outside active conversation")
	}

	reaction := &dbModel.Reaction{
		ID:               uuid.New().String(),
		MessageID:        messageID.String(),
		UserID:           userID.String(),
		ReactionType:     reactionType,
		ReactionEmoji:    emoji,
		ReactionSkinTone: skinTone,
		CreatedAt:        time.Now(),
	}

	if dbErr := s.reactionRepo.AddReaction(ctx, reaction); dbErr != nil {
		return dbErr
	}

	// Publish Event
	event := &domain.MessageEvent{
		EventType:      domain.MessageEventReactionAdded,
		MessageID:      messageID,
		ConversationID: uuid.MustParse(msg.ConversationID),
		SenderUserID:   userID,
		Timestamp:      time.Now(),
		Payload:        reaction, // Send the full reaction details as payload
	}
	if pubErr := s.eventPublisher.PublishMessageEvent(ctx, event); pubErr != nil {
		s.logger.Error("Failed to publish reaction event", logger.Error(pubErr))
	}

	return nil
}

func (s *reactionService) RemoveReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reactionType string) pkgErrors.AppError {
	// First get message context for event publishing
	msg, err := s.messageRepo.GetMessageByID(ctx, messageID.String())
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeNotFound, "Message not found")
	}

	if dbErr := s.reactionRepo.RemoveReaction(ctx, messageID, userID, reactionType); dbErr != nil {
		return dbErr
	}

	// Publish Event
	event := &domain.MessageEvent{
		EventType:      domain.MessageEventReactionRemoved,
		MessageID:      messageID,
		ConversationID: uuid.MustParse(msg.ConversationID),
		SenderUserID:   userID,
		Timestamp:      time.Now(),
		Payload: map[string]interface{}{
			"reaction_type": reactionType,
		},
	}
	if pubErr := s.eventPublisher.PublishMessageEvent(ctx, event); pubErr != nil {
		s.logger.Error("Failed to publish reaction removal event", logger.Error(pubErr))
	}

	return nil
}

func (s *reactionService) GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]dbModel.Reaction, pkgErrors.AppError) {
	return s.reactionRepo.GetMessageReactions(ctx, messageID)
}

func (s *reactionService) GetMessageReactionsWithSummary(ctx context.Context, messageID uuid.UUID, requesterUserID uuid.UUID) ([]dbModel.Reaction, []ReactionSummaryEntry, pkgErrors.AppError) {
	reactions, err := s.reactionRepo.GetMessageReactions(ctx, messageID)
	if err != nil {
		return nil, nil, err
	}

	counts := make(map[string]int)
	reactedByMe := make(map[string]bool)
	order := make([]string, 0)
	requesterID := requesterUserID.String()

	for _, r := range reactions {
		if _, seen := counts[r.ReactionType]; !seen {
			order = append(order, r.ReactionType)
		}
		counts[r.ReactionType]++
		if r.UserID == requesterID {
			reactedByMe[r.ReactionType] = true
		}
	}

	summary := make([]ReactionSummaryEntry, 0, len(order))
	for _, rt := range order {
		summary = append(summary, ReactionSummaryEntry{
			ReactionType: rt,
			Count:        counts[rt],
			ReactedByMe:  reactedByMe[rt],
		})
	}

	return reactions, summary, nil
}
