package repo

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	msgErr "echo-backend/services/message-service/internal/error"
	"errors"
	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModel "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"

	"github.com/google/uuid"
)

type ConversationRepository interface {
	ConversationExists(ctx context.Context, conversationID uuid.UUID) (bool, pkgErrors.AppError)
	GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, pkgErrors.AppError)
	GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]domain.ConversationParticipant, pkgErrors.AppError)
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, pkgErrors.AppError)
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

func (r *conversationRepository) GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, pkgErrors.AppError) {
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
		SELECT cp.*, COALESCE(p.display_name, p.username, '') as user_name, COALESCE(p.avatar_url, '') as avatar_url
		FROM messages.conversation_participants cp
		LEFT JOIN users.profiles p ON cp.user_id = p.user_id
		WHERE cp.conversation_id = $1
	`
	rows, dbErr := r.db.Query(ctx, participantsQuery, conversationID)
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
		var avatarURL, userName string
		if scanErr := rows.ScanModelAndAppend(&cp, &avatarURL, &userName); scanErr != nil {
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
		dbParticipants = append(dbParticipants, *domain.NewConversationParticipant(cp, userName, avatarURL))
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

func (r *conversationRepository) GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]domain.ConversationParticipant, pkgErrors.AppError) {
	query := `
		SELECT
			cp.id,
			cp.conversation_id,
			cp.user_id,
			cp.role,
			cp.joined_at,
			cp.left_at,
			cp.removed_at,
			cp.last_read_at,
			cp.unread_count,
			cp.is_muted,
			cp.is_pinned,
			cp.is_archived,
			cp.can_send_messages,
			cp.can_send_media,
			cp.can_add_members,
			cp.can_edit_info,
			cp.can_delete_messages,
			COALESCE(p.display_name, p.username, '') as user_name,
			COALESCE(p.avatar_url, '') as avatar_url
		FROM messages.conversation_participants cp
		LEFT JOIN users.profiles p ON cp.user_id = p.user_id
		WHERE cp.conversation_id = $1
	`
	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		r.logger.Error("Failed to get conversation participants",
			logger.Error(err),
			logger.String("conversation_id", conversationID.String()),
		)
		return nil, pkgErrors.FromError(
			err,
			pkgErrors.CodeDatabaseError,
			"Failed to get conversation participants",
		)
	}
	defer rows.Close()

	var participants []domain.ConversationParticipant
	for rows.Next() {
		var cp dbModel.ConversationParticipant
		var userName, avatarURL string

		if scanErr := rows.Scan(
			&cp.ID,
			&cp.ConversationID,
			&cp.UserID,
			&cp.Role,
			&cp.JoinedAt,
			&cp.LeftAt,
			&cp.RemovedAt,
			&cp.LastReadAt,
			&cp.UnreadCount,
			&cp.IsMuted,
			&cp.IsPinned,
			&cp.IsArchived,
			&cp.CanSendMessages,
			&cp.CanSendMedia,
			&cp.CanAddMembers,
			&cp.CanEditInfo,
			&cp.CanDeleteMessages,
			&userName,
			&avatarURL,
		); scanErr != nil {
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

		participants = append(participants, *domain.NewConversationParticipant(cp, userName, avatarURL))
	}

	r.logger.Debug("Fetched conversation participants",
		logger.String("conversation_id", conversationID.String()),
		logger.Int("participant_count", len(participants)),
		logger.Any("participants", participants),
	)

	return participants, nil
}

func (r *conversationRepository) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, pkgErrors.AppError) {
	// get conversation IDs for the user
	query := `
		SELECT c.id
		FROM messages.conversations c
		JOIN messages.conversation_participants cp ON c.id = cp.conversation_id
		WHERE cp.user_id = $1
		ORDER BY c.updated_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to get user conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(
			err,
			pkgErrors.CodeDatabaseError,
			"Failed to get user conversations",
		)
	}
	defer rows.Close()

	var conversationIDs []string
	for rows.Next() {
		var convID string
		if scanErr := rows.Scan(&convID); scanErr != nil {
			r.logger.Error("Failed to scan conversation ID",
				logger.String("user_id", userID.String()),
				logger.Error(scanErr),
			)
			return nil, pkgErrors.FromError(
				scanErr,
				pkgErrors.CodeDatabaseError,
				"Failed to scan conversation ID",
			)
		}
		conversationIDs = append(conversationIDs, convID)
	}

	var domainConversations []domain.Conversation
	for _, convID := range conversationIDs {
		domainConv, dbErr := r.GetConversationByID(ctx, uuid.MustParse(convID))
		if dbErr != nil {
			r.logger.Error("Failed to get conversation by ID",
				logger.String("user_id", userID.String()),
				logger.String("conversation_id", convID),
				logger.Error(dbErr),
			)
			return nil, dbErr
		}
		domainConversations = append(domainConversations, *domainConv)
	}

	r.logger.Debug("Fetched user conversations",
		logger.String("user_id", userID.String()),
		logger.Int("conversation_count", len(domainConversations)),
		logger.Any("conversations", domainConversations),
	)

	return domainConversations, nil
}

func (r *conversationRepository) CreateConversation(ctx context.Context, input domain.CreateConversationInput) (conv *domain.Conversation, err pkgErrors.AppError) {
	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return nil, pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "Failed to create conversation")
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()

	if input.ConversationType == domain.ConversationTypeDirect {
		query := `
			SELECT c.id
			FROM messages.conversations c
			JOIN messages.conversation_participants cp1 ON c.id = cp1.conversation_id
			JOIN messages.conversation_participants cp2 ON c.id = cp2.conversation_id
			WHERE c.conversation_type = $1
			AND cp1.user_id = $2
			AND cp2.user_id = $3
			GROUP BY c.id
			HAVING COUNT(DISTINCT cp1.user_id) = 1 AND COUNT(DISTINCT cp2.user_id) = 1
		`
		var existingConversationID uuid.UUID
		dbErr := tx.QueryRow(ctx, query, input.ConversationType, input.CreatorUserID, input.ParticipantIDs[0]).Scan(&existingConversationID)
		if dbErr == nil {
			return r.GetConversationByID(ctx, existingConversationID)
		}
		if !postgres.IsNotFoundError(dbErr) {
			return nil, pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "Failed to check existing conversation")
		}
	}

	var conversationID uuid.UUID
	insertQuery := `
		INSERT INTO messages.conversations (
			conversation_type,
			title,
			description,
			avatar_url,
			creator_user_id,
			is_group,
			is_channel,
			is_encrypted,
			is_public,
			member_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	row := tx.QueryRow(
		ctx,
		insertQuery,
		input.ConversationType,
		input.Title,
		input.Description,
		input.AvatarURL,
		input.CreatorUserID,
		input.ConversationType == domain.ConversationTypeGroup,
		input.ConversationType == domain.ConversationTypeChannel,
		input.IsEncrypted,
		input.IsPublic,
		input.MemberCount,
	)
	if scanErr := row.Scan(&conversationID); scanErr != nil {
		return nil, pkgErrors.FromError(scanErr, msgErr.CodeConversationCreateFailed.String(), "failed to create conversation")
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
			if _, execErr := tx.Exec(ctx, participantQuery, conversationID, participantID, domain.ParticipantRoleMember); execErr != nil {
				return nil, pkgErrors.FromError(execErr, "PARTICIPANT_INSERT_FAILED", "failed to add participant")
			}
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, pkgErrors.FromError(commitErr, "TRANSACTION_COMMIT_FAILED", "failed to commit transaction")
	}

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
	rows, queryErr := r.db.Query(ctx, fetchQuery, conversationID)
	if queryErr != nil {
		return nil, pkgErrors.FromError(queryErr, pkgErrors.CodeInternal, "Failed to fetch conversation")
	}
	defer rows.Close()

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

		if scanErr := rows.Scan(
			&id, &conversationType, &title, &description, &avatarURL,
			&creatorUserID, &isEncrypted, &isPublic, &memberCount,
			&participantID, &role, &joinedAt,
		); scanErr != nil {
			return nil, pkgErrors.FromError(scanErr, pkgErrors.CodeInternal, "Failed to scan conversation")
		}

		if conv == nil {
			conv = &domain.Conversation{
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

	if conv == nil {
		return nil, pkgErrors.FromError(nil, pkgErrors.CodeNotFound, "Conversation not found after creation")
	}

	conv.Participants = participants
	return conv, nil
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
			logger.Bool("is_no_rows", postgres.IsNotFoundError(dbErr)),
		)
		if postgres.IsNotFoundError(dbErr) {
			err = pkgErrors.FromError(
				dbErr,
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
				errors.New("participant has left the conversation"),
				pkgErrors.CodePermissionDenied,
				"participant has left the conversation",
			)
		}
		if removedAt != nil && utils.IsValidTime(*removedAt) {
			err = pkgErrors.FromError(
				errors.New("participant has been removed from the conversation"),
				pkgErrors.CodePermissionDenied,
				"Participant has been removed from the conversation",
			)
		}
		if !canSendMessages {
			err = pkgErrors.FromError(
				errors.New("participant is not allowed to send messages"),
				pkgErrors.CodePermissionDenied,
				"Participant is not allowed to send messages",
			)
		}
		if !isActive {
			err = pkgErrors.FromError(
				errors.New("conversation is inactive"),
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
