package repo

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	"fmt"
	"strings"
	"time"

	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// UpdateConversation updates conversation fields dynamically
func (r *conversationRepository) UpdateConversation(ctx context.Context, conversationID uuid.UUID, input domain.UpdateConversationInput) pkgErrors.AppError {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if input.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *input.Title)
		argIdx++
	}
	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}
	if input.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *input.AvatarURL)
		argIdx++
	}
	if input.IsPublic != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_public = $%d", argIdx))
		args = append(args, *input.IsPublic)
		argIdx++
	}
	if input.WhoCanSendMessages != nil {
		setClauses = append(setClauses, fmt.Sprintf("who_can_send_messages = $%d", argIdx))
		args = append(args, *input.WhoCanSendMessages)
		argIdx++
	}
	if input.WhoCanAddMembers != nil {
		setClauses = append(setClauses, fmt.Sprintf("who_can_add_members = $%d", argIdx))
		args = append(args, *input.WhoCanAddMembers)
		argIdx++
	}
	if input.WhoCanEditInfo != nil {
		setClauses = append(setClauses, fmt.Sprintf("who_can_edit_info = $%d", argIdx))
		args = append(args, *input.WhoCanEditInfo)
		argIdx++
	}
	if input.WhoCanPinMessages != nil {
		setClauses = append(setClauses, fmt.Sprintf("who_can_pin_messages = $%d", argIdx))
		args = append(args, *input.WhoCanPinMessages)
		argIdx++
	}
	if input.InviteLinkExpiresAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("invite_link_expires_at = $%d", argIdx))
		args = append(args, *input.InviteLinkExpiresAt)
		argIdx++
	}
	if input.JoinApprovalRequired != nil {
		setClauses = append(setClauses, fmt.Sprintf("join_approval_required = $%d", argIdx))
		args = append(args, *input.JoinApprovalRequired)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, conversationID)

	query := fmt.Sprintf(
		"UPDATE messages.conversations SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		argIdx,
	)

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to update conversation",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update conversation")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "conversation not found")
	}
	return nil
}

// SoftDeleteConversation soft-deletes a conversation
func (r *conversationRepository) SoftDeleteConversation(ctx context.Context, conversationID uuid.UUID) pkgErrors.AppError {
	query := `
		UPDATE messages.conversations
		SET is_active = FALSE, deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(ctx, query, conversationID)
	if err != nil {
		r.logger.Error("failed to soft delete conversation",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete conversation")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "conversation not found or already deleted")
	}
	return nil
}

// AddParticipants adds participants to a conversation
func (r *conversationRepository) AddParticipants(ctx context.Context, conversationID uuid.UUID, participantIDs []uuid.UUID, invitedByUserID uuid.UUID) pkgErrors.AppError {
	if len(participantIDs) == 0 {
		return nil
	}

	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to begin transaction")
	}

	var txErr *database.DBError
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if txErr != nil {
			tx.Rollback()
		}
	}()

	query := `
		INSERT INTO messages.conversation_participants (
			id, conversation_id, user_id, role, join_method, invited_by_user_id, joined_at
		) VALUES ($1, $2, $3, 'member', 'invited', $4, NOW())
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET
			left_at = NULL, removed_at = NULL, removed_by_user_id = NULL,
			role = 'member', updated_at = NOW()
		WHERE messages.conversation_participants.left_at IS NOT NULL
		   OR messages.conversation_participants.removed_at IS NOT NULL
	`

	for _, participantID := range participantIDs {
		id := uuid.New()
		_, txErr = tx.Exec(ctx, query, id, conversationID, participantID, invitedByUserID)
		if txErr != nil {
			r.logger.Error("failed to add participant",
				logger.String("conversation_id", conversationID.String()),
				logger.String("participant_id", participantID.String()),
				logger.Error(txErr),
			)
			return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to add participant")
		}
	}

	// Update member count
	countQuery := `
		UPDATE messages.conversations
		SET member_count = (
			SELECT COUNT(*) FROM messages.conversation_participants
			WHERE conversation_id = $1 AND left_at IS NULL AND removed_at IS NULL
		), updated_at = NOW()
		WHERE id = $1
	`
	_, txErr = tx.Exec(ctx, countQuery, conversationID)
	if txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to update member count")
	}

	if txErr = tx.Commit(); txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeInternal, "failed to commit add participants transaction")
	}
	return nil
}

// RemoveParticipant removes a participant from a conversation
func (r *conversationRepository) RemoveParticipant(ctx context.Context, conversationID uuid.UUID, participantID uuid.UUID, removedByUserID uuid.UUID) pkgErrors.AppError {
	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to begin transaction")
	}

	var txErr *database.DBError
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if txErr != nil {
			tx.Rollback()
		}
	}()

	query := `
		UPDATE messages.conversation_participants
		SET removed_at = NOW(), removed_by_user_id = $3, updated_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL AND removed_at IS NULL
	`
	result, txErr := tx.Exec(ctx, query, conversationID, participantID, removedByUserID)
	if txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to remove participant")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		txErr = &database.DBError{}
		return pkgErrors.New(pkgErrors.CodeNotFound, "participant not found in conversation")
	}

	// Update member count
	countQuery := `
		UPDATE messages.conversations
		SET member_count = (
			SELECT COUNT(*) FROM messages.conversation_participants
			WHERE conversation_id = $1 AND left_at IS NULL AND removed_at IS NULL
		), updated_at = NOW()
		WHERE id = $1
	`
	_, txErr = tx.Exec(ctx, countQuery, conversationID)
	if txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to update member count")
	}

	if txErr = tx.Commit(); txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeInternal, "failed to commit remove participant transaction")
	}
	return nil
}

// UpdateParticipantRole updates a participant's role
func (r *conversationRepository) UpdateParticipantRole(ctx context.Context, conversationID uuid.UUID, participantID uuid.UUID, role string) pkgErrors.AppError {
	query := `
		UPDATE messages.conversation_participants
		SET role = $3, updated_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL AND removed_at IS NULL
	`
	result, err := r.db.Exec(ctx, query, conversationID, participantID, role)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update participant role")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "participant not found in conversation")
	}
	return nil
}

// MuteConversation mutes a conversation for a user
func (r *conversationRepository) MuteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, muteUntil *time.Time) pkgErrors.AppError {
	query := `
		UPDATE messages.conversation_participants
		SET is_muted = TRUE, muted_until = $3, updated_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL AND removed_at IS NULL
	`
	result, err := r.db.Exec(ctx, query, conversationID, userID, muteUntil)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to mute conversation")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "participant not found in conversation")
	}
	return nil
}

// UnmuteConversation unmutes a conversation for a user
func (r *conversationRepository) UnmuteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) pkgErrors.AppError {
	query := `
		UPDATE messages.conversation_participants
		SET is_muted = FALSE, muted_until = NULL, updated_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL AND removed_at IS NULL
	`
	result, err := r.db.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to unmute conversation")
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "participant not found in conversation")
	}
	return nil
}

// GetUnreadCount returns the unread count for a user in a conversation
func (r *conversationRepository) GetUnreadCount(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (int, pkgErrors.AppError) {
	query := `
		SELECT COALESCE(unread_count, 0)
		FROM messages.conversation_participants
		WHERE conversation_id = $1 AND user_id = $2
	`
	var count int
	err := r.db.QueryRow(ctx, query, conversationID, userID).Scan(&count)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get unread count")
	}
	return count, nil
}

// GetConversationType returns the conversation type
func (r *conversationRepository) GetConversationType(ctx context.Context, conversationID uuid.UUID) (string, pkgErrors.AppError) {
	query := `SELECT conversation_type FROM messages.conversations WHERE id = $1`
	var convType string
	err := r.db.QueryRow(ctx, query, conversationID).Scan(&convType)
	if err != nil {
		return "", pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get conversation type")
	}
	return convType, nil
}
