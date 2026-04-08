package repo

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	msgErr "echo-backend/services/message-service/internal/error"
	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// GetUserConversations retrieves all active conversations for a user, with participant data and viewer-scoped metadata.
func (r *conversationRepository) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, pkgErrors.AppError) {
	rows, err := r.db.Query(ctx, `
		SELECT
			c.id,
			c.conversation_type,
			c.title,
			c.description,
			c.avatar_url,
			c.creator_user_id,
			c.is_encrypted,
			c.is_public,
			c.message_count,
			c.last_message_id,
			c.last_message_at,
			c.last_activity_at,
			c.created_at,
			c.updated_at,
			COUNT(cp_all.user_id) FILTER (
				WHERE cp_all.left_at IS NULL AND cp_all.removed_at IS NULL
			) AS member_count,
			cp_viewer.role,
			cp_viewer.unread_count,
			cp_viewer.mention_count,
			cp_viewer.is_muted,
			cp_viewer.is_pinned,
			cp_viewer.joined_at
		FROM messages.conversations c
		JOIN messages.conversation_participants cp_viewer
			ON  cp_viewer.conversation_id = c.id
			AND cp_viewer.user_id         = $1
			AND cp_viewer.left_at         IS NULL
			AND cp_viewer.removed_at      IS NULL
		JOIN messages.conversation_participants cp_all
			ON cp_all.conversation_id = c.id
		WHERE c.is_active  = TRUE
		  AND c.deleted_at IS NULL
		GROUP BY
			c.id, c.conversation_type, c.title, c.description, c.avatar_url,
			c.creator_user_id, c.is_encrypted, c.is_public, c.message_count,
			c.last_message_id, c.last_message_at, c.last_activity_at,
			c.created_at, c.updated_at,
			cp_viewer.role, cp_viewer.unread_count, cp_viewer.mention_count,
			cp_viewer.is_muted, cp_viewer.is_pinned, cp_viewer.joined_at
		ORDER BY COALESCE(c.last_activity_at, c.created_at) DESC
	`, userID)
	if err != nil {
		r.logger.Error("Failed to get user conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get user conversations")
	}
	defer rows.Close()

	var (
		conversations []domain.Conversation
		convIDs       []string
		convIndex     = make(map[uuid.UUID]int)
	)

	for rows.Next() {
		var c domain.Conversation
		if scanErr := rows.Scan(
			&c.ID,
			&c.ConversationType,
			&c.Title,
			&c.Description,
			&c.AvatarURL,
			&c.CreatorUserID,
			&c.IsEncrypted,
			&c.IsPublic,
			&c.MessageCount,
			&c.LastMessageID,
			&c.LastMessageAt,
			&c.LastActivityAt,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.MemberCount,
			&c.ViewerRole,
			&c.UnreadCount,
			&c.MentionCount,
			&c.IsMuted,
			&c.IsPinned,
			&c.JoinedAt,
		); scanErr != nil {
			r.logger.Error("Failed to scan conversation",
				logger.String("user_id", userID.String()),
				logger.Error(scanErr),
			)
			return nil, pkgErrors.FromError(scanErr, pkgErrors.CodeDatabaseError, "Failed to scan conversation")
		}
		convIndex[c.ID] = len(conversations)
		convIDs = append(convIDs, c.ID.String())
		conversations = append(conversations, c)
	}
	if err := rows.Err(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Row iteration error")
	}

	if len(convIDs) == 0 {
		return conversations, nil
	}

	pRows, err := r.db.Query(ctx, `
		SELECT
			cp.id,
			cp.conversation_id,
			cp.user_id,
			cp.role,
			cp.joined_at,
			cp.left_at,
			cp.last_read_at,
			cp.unread_count,
			cp.is_muted,
			cp.is_pinned,
			cp.is_archived,
			cp.can_send_messages,
			cp.can_add_members,
			cp.can_edit_info,
			cp.can_delete_messages,
			COALESCE(p.username,              ''),
			COALESCE(p.display_name,          ''),
			COALESCE(p.first_name,            ''),
			COALESCE(p.last_name,             ''),
			COALESCE(p.avatar_url,            ''),
			COALESCE(p.avatar_thumbnail_url,  ''),
			COALESCE(p.online_status,         'offline'),
			p.last_seen_at,
			COALESCE(p.is_verified,           FALSE)
		FROM messages.conversation_participants cp
		LEFT JOIN users.profiles p ON p.user_id = cp.user_id
		WHERE cp.conversation_id = ANY($1::uuid[])
		AND cp.left_at         IS NULL
		AND cp.removed_at      IS NULL
	`, pq.Array(convIDs))
	if err != nil {
		r.logger.Error("Failed to get participants",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get participants")
	}
	defer pRows.Close()

	for pRows.Next() {
		var p domain.ConversationParticipant
		if scanErr := pRows.Scan(
			&p.ID,
			&p.ConversationID,
			&p.UserID,
			&p.Role,
			&p.JoinedAt,
			&p.LeftAt,
			&p.LastReadAt,
			&p.UnreadCount,
			&p.IsMuted,
			&p.IsPinned,
			&p.IsArchived,
			&p.CanSendMessages,
			&p.CanAddMembers,
			&p.CanEditInfo,
			&p.CanDeleteMessages,
			&p.Username,
			&p.DisplayName,
			&p.FirstName,
			&p.LastName,
			&p.AvatarURL,
			&p.AvatarThumbnailURL,
			&p.OnlineStatus,
			&p.LastSeenAt,
			&p.IsVerified,
		); scanErr != nil {
			r.logger.Error("Failed to scan participant",
				logger.String("user_id", userID.String()),
				logger.Error(scanErr),
			)
			return nil, pkgErrors.FromError(scanErr, pkgErrors.CodeDatabaseError, "Failed to scan participant")
		}
		idx := convIndex[p.ConversationID]
		conversations[idx].Participants = append(conversations[idx].Participants, p)
	}
	if err := pRows.Err(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Participant row iteration error")
	}

	r.logger.Debug("Fetched user conversations",
		logger.String("user_id", userID.String()),
		logger.Int("count", len(conversations)),
	)

	return conversations, nil
}

// CreateConversation creates a new conversation (with duplicate-DM detection) and adds participants.
func (r *conversationRepository) CreateConversation(ctx context.Context, input domain.CreateConversationInput) (conv *domain.Conversation, err pkgErrors.AppError) {
	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return nil, pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "Failed to begin transaction")
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
		if len(input.ParticipantIDs) == 0 {
			return nil, pkgErrors.FromError(nil, pkgErrors.CodeBadRequest, "Direct conversation requires at least one participant")
		}
		var existingID uuid.UUID
		var otherParticipantID uuid.UUID
		if len(input.ParticipantIDs) > 0 {
			otherParticipantID = input.ParticipantIDs[0]
		} else {
			return nil, pkgErrors.FromError(nil, pkgErrors.CodeBadRequest, "Direct conversation requires a participant")
		}
		dbErr := tx.QueryRow(ctx, `
			SELECT c.id
			FROM messages.conversations c
			JOIN messages.conversation_participants cp1 ON cp1.conversation_id = c.id
			JOIN messages.conversation_participants cp2 ON cp2.conversation_id = c.id
			WHERE c.conversation_type = $1
			  AND cp1.user_id     = $2
			  AND cp2.user_id     = $3
			  AND cp1.left_at     IS NULL AND cp1.removed_at IS NULL
			  AND cp2.left_at     IS NULL AND cp2.removed_at IS NULL
			GROUP BY c.id
			HAVING COUNT(DISTINCT cp1.user_id) = 1 AND COUNT(DISTINCT cp2.user_id) = 1
		`, input.ConversationType, input.CreatorUserID, otherParticipantID).Scan(&existingID)
		if dbErr == nil {
			_ = tx.Rollback()
			return r.GetConversationByID(ctx, existingID, &input.CreatorUserID)
		}
		if !postgres.IsNotFoundError(dbErr) {
			return nil, pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "Failed to check for existing conversation")
		}
	}

	input = applyConversationDefaults(input)

	var conversationID uuid.UUID
	if scanErr := tx.QueryRow(ctx, `
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
			max_members,
			join_approval_required,
			who_can_send_messages,
			who_can_add_members,
			who_can_edit_info,
			who_can_pin_messages
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id
	`,
		input.ConversationType,
		input.Title,
		input.Description,
		input.AvatarURL,
		input.CreatorUserID,
		input.ConversationType == domain.ConversationTypeGroup,
		input.ConversationType == domain.ConversationTypeChannel,
		input.IsEncrypted,
		input.IsPublic,
		input.MaxMembers,
		input.JoinApprovalRequired,
		input.WhoCanSendMessages,
		input.WhoCanAddMembers,
		input.WhoCanEditInfo,
		input.WhoCanPinMessages,
	).Scan(&conversationID); scanErr != nil {
		return nil, pkgErrors.FromError(scanErr, msgErr.CodeConversationCreateFailed.String(), "Failed to create conversation")
	}

	if len(input.ParticipantIDs) > 0 {
		for _, participantID := range input.ParticipantIDs {
			if _, execErr := tx.Exec(ctx, `
				INSERT INTO messages.conversation_participants (
					conversation_id, user_id, role, join_method, joined_at
				) VALUES ($1, $2, $3, 'added', NOW())
				ON CONFLICT (conversation_id, user_id) DO NOTHING
			`, conversationID, participantID, domain.ParticipantRoleMember); execErr != nil {
				return nil, pkgErrors.FromError(execErr, "PARTICIPANT_INSERT_FAILED", "Failed to add participant")
			}
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, pkgErrors.FromError(commitErr, "TRANSACTION_COMMIT_FAILED", "Failed to commit transaction")
	}

	return r.GetConversationByID(ctx, conversationID, &input.CreatorUserID)
}

// applyConversationDefaults sets default permission values based on conversation type.
func applyConversationDefaults(input domain.CreateConversationInput) domain.CreateConversationInput {
	switch input.ConversationType {
	case domain.ConversationTypeDirect:
		input.WhoCanSendMessages = "all"
		input.WhoCanAddMembers = "admins"
		input.WhoCanEditInfo = "admins"
		input.WhoCanPinMessages = "admins"
		input.IsEncrypted = true

	case domain.ConversationTypeGroup:
		if input.WhoCanSendMessages == "" {
			input.WhoCanSendMessages = "all"
		}
		if input.WhoCanAddMembers == "" {
			input.WhoCanAddMembers = "admins"
		}
		if input.WhoCanEditInfo == "" {
			input.WhoCanEditInfo = "admins"
		}
		if input.WhoCanPinMessages == "" {
			input.WhoCanPinMessages = "admins"
		}
		if input.MaxMembers == nil {
			n := 256
			input.MaxMembers = &n
		}

	case domain.ConversationTypeChannel, domain.ConversationTypeBroadcast:
		input.WhoCanSendMessages = "admins"
		input.WhoCanAddMembers = "admins"
		input.WhoCanEditInfo = "admins"
		input.WhoCanPinMessages = "admins"
	}

	return input
}
