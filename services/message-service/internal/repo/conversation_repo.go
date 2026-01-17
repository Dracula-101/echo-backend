package repo

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	msgErr "echo-backend/services/message-service/internal/error"
	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"

	"github.com/google/uuid"
)

type ConversationRepository interface {
	CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, pkgErrors.AppError)
	ValidateConversationParticipant(conversationID uuid.UUID, userID uuid.UUID) pkgErrors.AppError
}

type conversationRepository struct {
	db     database.Database
	cache  cache.Cache
	logger logger.Logger
}

func NewConversationRepository(db database.Database, cache cache.Cache, logger logger.Logger) ConversationRepository {
	return &conversationRepository{db: db, cache: cache, logger: logger}
}

func (r *conversationRepository) CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, pkgErrors.AppError) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "Failed to create conversation")
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	var conversationID = uuid.Nil
	conversationQuery := `
		INSERT INTO messages.conversations (
			conversation_type,
			title,
			description,
			avatar_url,
			creator_user_id,
			is_group,
			is_channel,
			is_encrypted,
			is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	row := tx.QueryRow(
		ctx,
		conversationQuery,
		input.ConversationType,
		input.Title,
		input.Description,
		input.AvatarURL,
		input.CreatorUserID,
		input.ConversationType == domain.ConversationTypeGroup,
		input.ConversationType == domain.ConversationTypeChannel,
		input.IsEncrypted,
		input.IsPublic,
	)

	if err := row.Scan(&conversationID); err != nil {
		return nil, pkgErrors.FromError(err, msgErr.CodeConversationCreateFailed.String(), "failed to create conversation")
	}

	if len(input.ParticipantIDs) > 0 {
		participantQuery := `
			INSERT INTO messages.conversation_participants (
				conversation_id,
				user_id,
				role,
				joined_at
			) VALUES ($1, $2, $3, NOW())
		`

		for _, participantID := range input.ParticipantIDs {
			if _, err := tx.Exec(ctx, participantQuery, conversationID, participantID, domain.ParticipantRoleMember); err != nil {
				return nil, pkgErrors.FromError(err, "PARTICIPANT_INSERT_FAILED", "failed to add participant")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, pkgErrors.FromError(err, "TRANSACTION_COMMIT_FAILED", "failed to commit transaction")
	}

	// Fetch the conversation with participants using JOIN
	fetchQuery := `
		SELECT 
			c.id,
			c.conversation_type,
			c.title,
			c.description,
			c.avatar_url,
			c.creator_user_id,
			c.is_encrypted,
			c.is_public,
			COALESCE(COUNT(cp.user_id), 0) as member_count,
			cp.user_id,
			cp.role,
			cp.joined_at
		FROM messages.conversations c
		LEFT JOIN messages.conversation_participants cp ON c.id = cp.conversation_id
		WHERE c.id = $1
		GROUP BY c.id, c.conversation_type, c.title, c.description, c.avatar_url, 
				 c.creator_user_id, c.is_encrypted, c.is_public, cp.user_id, cp.role, cp.joined_at
	`

	rows, err := r.db.Query(ctx, fetchQuery, conversationID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "Failed to fetch conversation")
	}
	defer rows.Close()

	var conversation *domain.Conversation
	participants := make([]domain.ConversationParticipant, 0)

	for rows.Next() {
		var (
			id               uuid.UUID
			conversationType string
			title            *string
			description      *string
			avatarURL        *string
			creatorUserID    uuid.UUID
			isEncrypted      bool
			isPublic         bool
			memberCount      int
			participantID    *uuid.UUID
			role             *string
			joinedAt         *time.Time
		)

		if err := rows.Scan(
			&id, &conversationType, &title, &description, &avatarURL,
			&creatorUserID, &isEncrypted, &isPublic, &memberCount,
			&participantID, &role, &joinedAt,
		); err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "Failed to scan conversation")
		}

		if conversation == nil {
			conversation = &domain.Conversation{
				ID:               id,
				ConversationType: domain.ConversationType(conversationType),
				Title:            *title,
				Description:      *description,
				AvatarURL:        avatarURL,
				CreatorUserID:    creatorUserID,
				IsEncrypted:      isEncrypted,
				IsPublic:         isPublic,
				MemberCount:      memberCount,
				MessageCount:     0,
				UnreadCount:      0,
				LastMessage:      nil,
			}
		}

		if participantID != nil && role != nil {
			participants = append(participants, domain.ConversationParticipant{
				ConversationID: id,
				UserID:         *participantID,
				Role:           domain.ParticipantRole(*role),
				JoinedAt:       *joinedAt,
			})
		}
	}

	if conversation == nil {
		return nil, pkgErrors.FromError(nil, pkgErrors.CodeNotFound, "Conversation not found after creation")
	}

	conversation.Participants = participants
	return conversation, nil
}

func (r *conversationRepository) ValidateConversationParticipant(conversationID uuid.UUID, userID uuid.UUID) pkgErrors.AppError {
	query := `
		SELECT 
			cp.can_send_messages,
			cp.can_send_media,
			cp.left_at,
			cp.removed_at,
			c.is_active
		FROM messages.conversation_participants cp
		INNER JOIN messages.conversations c ON c.id = cp.conversation_id
		WHERE cp.conversation_id = $1 AND cp.user_id = $2
	`

	var canSendMessages, canSendMedia, isActive bool
	var leftAt, removedAt *time.Time

	dbErr := r.db.QueryRow(context.Background(), query, conversationID, userID).Scan(
		&canSendMessages,
		&canSendMedia,
		&leftAt,
		&removedAt,
		&isActive,
	)
	var err pkgErrors.AppError
	if dbErr != nil {
		r.logger.Error("Failed to validate conversation participant",
			logger.Error(dbErr),
			logger.String("conversation_id", conversationID.String()),
			logger.String("user_id", userID.String()),
			logger.Bool("is_no_rows", postgres.IsNoRowsError(dbErr)),
		)
		if postgres.IsNoRowsError(dbErr) {
			err = pkgErrors.FromError(
				msgErr.ParticipantNotInConversationError,
				pkgErrors.CodeNotFound,
				"Participant not in conversation",
			)
		} else {
			err = pkgErrors.FromError(
				dbErr,
				pkgErrors.CodeDatabaseError,
				"Failed to validate conversation participant",
			)
		}
	} else {
		if leftAt != nil && utils.IsValidTime(*leftAt) {
			err = pkgErrors.FromError(
				msgErr.ParticipantLeftConversationError,
				pkgErrors.CodePermissionDenied,
				"participant has left the conversation",
			)
		}
		if removedAt != nil && utils.IsValidTime(*removedAt) {
			err = pkgErrors.FromError(
				msgErr.ParticipantRemovedFromConversationError,
				pkgErrors.CodePermissionDenied,
				"Participant has been removed from the conversation",
			)
		}
		if !canSendMessages {
			err = pkgErrors.FromError(
				msgErr.ParticipantNotAllowedToSendMessagesError,
				pkgErrors.CodePermissionDenied,
				"Participant is not allowed to send messages",
			)
		}
		if !isActive {
			err = pkgErrors.FromError(
				msgErr.ConversationInactiveError,
				pkgErrors.CodePermissionDenied,
				"Conversation is inactive",
			)
		}
	}

	if err != nil {
		return err.
			WithService(msgErr.ServiceName).
			WithDetail("conversation_id", conversationID).
			WithDetail("user_id", userID)
	}

	return nil
}
