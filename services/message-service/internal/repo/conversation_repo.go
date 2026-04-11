package repo

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModel "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"time"

	"github.com/google/uuid"
)

// ConversationRepository handles conversation CRUD and participant operations.
type ConversationRepository interface {
	ConversationExists(ctx context.Context, conversationID uuid.UUID) (bool, pkgErrors.AppError)
	GetConversationByID(ctx context.Context, conversationID uuid.UUID, currentUserID *uuid.UUID) (*domain.Conversation, pkgErrors.AppError)
	GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]domain.ConversationParticipant, pkgErrors.AppError)
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, pkgErrors.AppError)
	CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, pkgErrors.AppError)
	ValidateConversationParticipant(conversationID uuid.UUID, userID uuid.UUID) pkgErrors.AppError
	UpdateConversation(ctx context.Context, conversationID uuid.UUID, input domain.UpdateConversationInput) pkgErrors.AppError
	SoftDeleteConversation(ctx context.Context, conversationID uuid.UUID) pkgErrors.AppError
	AddParticipants(ctx context.Context, conversationID uuid.UUID, participantIDs []uuid.UUID, invitedByUserID uuid.UUID) pkgErrors.AppError
	RemoveParticipant(ctx context.Context, conversationID uuid.UUID, participantID uuid.UUID, removedByUserID uuid.UUID) pkgErrors.AppError
	UpdateParticipantRole(ctx context.Context, conversationID uuid.UUID, participantID uuid.UUID, role string) pkgErrors.AppError
	MuteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, muteUntil *time.Time) pkgErrors.AppError
	UnmuteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) pkgErrors.AppError
	GetUnreadCount(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (int, pkgErrors.AppError)
	GetConversationType(ctx context.Context, conversationID uuid.UUID) (string, pkgErrors.AppError)
}

type conversationRepository struct {
	db     database.Database
	cache  cache.Cache
	logger logger.Logger
}

func NewConversationRepository(db database.Database, cache cache.Cache, logger logger.Logger) ConversationRepository {
	return &conversationRepository{db: db, cache: cache, logger: logger}
}

func (r *conversationRepository) ConversationExists(ctx context.Context, conversationID uuid.UUID) (bool, pkgErrors.AppError) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM messages.conversations
			WHERE id = $1
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, conversationID).Scan(&exists)
	if err != nil {
		r.logger.Error("Failed to check if conversation exists",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return false, pkgErrors.FromError(
			err,
			pkgErrors.CodeDatabaseError,
			"Failed to check if conversation exists",
		)
	}

	return exists, nil
}

func (r *conversationRepository) GetConversationByID(ctx context.Context, conversationID uuid.UUID, currentUserID *uuid.UUID) (*domain.Conversation, pkgErrors.AppError) {
	query := `
		SELECT * FROM messages.conversations
		WHERE id = $1
	`
	var conv dbModel.Conversation
	err := r.db.QueryRow(ctx, query, conversationID)
	if scanErr := err.ScanModel(&conv); scanErr != nil {
		if postgres.IsNotFoundError(scanErr) {
			return nil, pkgErrors.FromError(
				scanErr,
				pkgErrors.CodeNotFound,
				"Conversation not found",
			)
		}
		r.logger.Error("Failed to get conversation by ID",
			logger.Error(scanErr),
			logger.String("conversation_id", conversationID.String()),
			logger.Bool("is_no_rows", postgres.IsNotFoundError(scanErr)),
		)
		return nil, pkgErrors.FromError(
			scanErr,
			pkgErrors.CodeDatabaseError,
			"Failed to get conversation by ID",
		)
	}
	// join from users.profiles to get participant details
	participantsQuery := `
		SELECT
			cp.*,
			COALESCE(p.display_name, p.username, '') AS user_name,
			COALESCE(p.avatar_url, '') AS avatar_url
		FROM messages.conversation_participants cp
		LEFT JOIN users.profiles p ON cp.user_id = p.user_id
		WHERE cp.conversation_id = $1
		AND cp.user_id <> $2
	`
	rows, dbErr := r.db.Query(ctx, participantsQuery, conversationID, currentUserID)
	if dbErr != nil {
		r.logger.Error("Failed to get conversation participants",
			logger.Error(dbErr),
			logger.String("conversation_id", conversationID.String()),
		)
		return nil, pkgErrors.FromError(
			dbErr,
			pkgErrors.CodeDatabaseError,
			"Failed to get conversation participants",
		)
	}
	defer rows.Close()

	var dbParticipants []domain.ConversationParticipant
	for rows.Next() {
		var cp dbModel.ConversationParticipant
		var userName, avatarURL, firstName, lastName, onlineStatus string
		var lastSeenAt time.Time
		var isVerified bool
		if scanErr := rows.ScanModelAndAppend(&cp, &userName, &avatarURL); scanErr != nil {
			r.logger.Error("Failed to scan conversation participant",
				logger.Error(scanErr),
				logger.String("conversation_id", conversationID.String()),
			)
			return nil, pkgErrors.FromError(
				scanErr,
				pkgErrors.CodeDatabaseError,
				"Failed to scan conversation participant",
			)
		}
		dbParticipants = append(dbParticipants, *domain.NewConversationParticipant(cp, userName, avatarURL, firstName, lastName, avatarURL, avatarURL, onlineStatus, &lastSeenAt, isVerified))
	}

	lastMessageQuery := `
		SELECT * FROM messages.messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var lastMsg dbModel.Message
	row := r.db.QueryRow(ctx, lastMessageQuery, conversationID)
	if scanErr := row.ScanModel(&lastMsg); scanErr != nil {
		if !postgres.IsNotFoundError(scanErr) {
			r.logger.Error("Failed to get last message for conversation",
				logger.Error(scanErr),
				logger.String("conversation_id", conversationID.String()),
			)
			return nil, pkgErrors.FromError(
				scanErr,
				pkgErrors.CodeDatabaseError,
				"Failed to get last message for conversation",
			)
		} else {
			return domain.NewConversation(conv, dbParticipants, nil), nil
		}
	}
	lastMsgDomain := domain.NewMessage(lastMsg)
	return domain.NewConversation(conv, dbParticipants, &lastMsgDomain), nil
}
