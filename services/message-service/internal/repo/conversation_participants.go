package repo

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	msgErr "echo-backend/services/message-service/internal/error"
	"errors"
	"shared/pkg/database/postgres"
	dbModel "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"

	"github.com/google/uuid"
)

// GetConversationParticipants retrieves all participants for a conversation with enriched profile data.
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
			COALESCE(p.avatar_url, '') as avatar_url,
			COALESCE(p.first_name, '') as first_name,
			COALESCE(p.last_name, '') as last_name,
			COALESCE(p.online_status, 'offline') as online_status,
			COALESCE(p.last_seen_at, '1970-01-01T00:00:00Z') as last_seen_at,
			COALESCE(p.is_verified, false) as is_verified
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
		var userName, avatarURL, firstName, lastName, onlineStatus string
		var lastSeenAt time.Time
		var isVerified bool

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
			&firstName,
			&lastName,
			&onlineStatus,
			&lastSeenAt,
			&isVerified,
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

		participants = append(participants, *domain.NewConversationParticipant(cp, userName, userName, firstName, lastName, avatarURL, avatarURL, onlineStatus, &lastSeenAt, isVerified))
	}

	r.logger.Debug("Fetched conversation participants",
		logger.String("conversation_id", conversationID.String()),
		logger.Int("participant_count", len(participants)),
		logger.Any("participants", participants),
	)

	return participants, nil
}

// ValidateConversationParticipant checks if a user is a valid, active participant who can send messages.
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
