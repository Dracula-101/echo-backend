package repo

import (
	"context"
	"shared/pkg/database"
	"shared/pkg/logger"
	"time"

	"github.com/google/uuid"
)

// Invite represents a row in messages.conversation_invites.
type Invite struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	InviterUserID  uuid.UUID  `json:"inviter_user_id"`
	InviteCode     string     `json:"invite_code"`
	Status         string     `json:"status"`
	MaxUses        int        `json:"max_uses"`
	UseCount       int        `json:"use_count"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// InviteRepository handles DB access for conversation invites.
type InviteRepository interface {
	CreateInvite(ctx context.Context, conversationID, inviterUserID uuid.UUID, inviteCode string, maxUses int, expiresAt *time.Time) (*Invite, error)
	GetInviteByCode(ctx context.Context, code string) (*Invite, error)
	GetConversationInvites(ctx context.Context, conversationID uuid.UUID) ([]Invite, error)
	IncrementUseCount(ctx context.Context, inviteID uuid.UUID) error
	RevokeInvite(ctx context.Context, inviteID uuid.UUID) error
}

type inviteRepository struct {
	db     database.Database
	logger logger.Logger
}

func NewInviteRepository(db database.Database, log logger.Logger) InviteRepository {
	return &inviteRepository{db: db, logger: log}
}

func (r *inviteRepository) CreateInvite(ctx context.Context, conversationID, inviterUserID uuid.UUID, inviteCode string, maxUses int, expiresAt *time.Time) (*Invite, error) {
	query := `
		INSERT INTO messages.conversation_invites (conversation_id, inviter_user_id, invite_code, max_uses, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, conversation_id, inviter_user_id, invite_code, status, max_uses, use_count, expires_at, created_at
	`

	var invite Invite
	err := r.db.QueryRow(ctx, query, conversationID, inviterUserID, inviteCode, maxUses, expiresAt).Scan(
		&invite.ID, &invite.ConversationID, &invite.InviterUserID, &invite.InviteCode,
		&invite.Status, &invite.MaxUses, &invite.UseCount, &invite.ExpiresAt, &invite.CreatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create invite",
			logger.String("conversation_id", conversationID.String()),
			logger.String("inviter_user_id", inviterUserID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	return &invite, nil
}

func (r *inviteRepository) GetInviteByCode(ctx context.Context, code string) (*Invite, error) {
	query := `
		SELECT id, conversation_id, inviter_user_id, invite_code, status, max_uses, use_count, expires_at, created_at
		FROM messages.conversation_invites
		WHERE invite_code = $1
	`

	var invite Invite
	err := r.db.QueryRow(ctx, query, code).Scan(
		&invite.ID, &invite.ConversationID, &invite.InviterUserID, &invite.InviteCode,
		&invite.Status, &invite.MaxUses, &invite.UseCount, &invite.ExpiresAt, &invite.CreatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to get invite by code",
			logger.String("invite_code", code),
			logger.Error(err),
		)
		return nil, err
	}

	return &invite, nil
}

func (r *inviteRepository) GetConversationInvites(ctx context.Context, conversationID uuid.UUID) ([]Invite, error) {
	query := `
		SELECT id, conversation_id, inviter_user_id, invite_code, status, max_uses, use_count, expires_at, created_at
		FROM messages.conversation_invites
		WHERE conversation_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		r.logger.Error("Failed to get conversation invites",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var invites []Invite
	for rows.Next() {
		var invite Invite
		if err := rows.Scan(
			&invite.ID, &invite.ConversationID, &invite.InviterUserID, &invite.InviteCode,
			&invite.Status, &invite.MaxUses, &invite.UseCount, &invite.ExpiresAt, &invite.CreatedAt,
		); err != nil {
			r.logger.Error("Failed to scan invite row",
				logger.String("conversation_id", conversationID.String()),
				logger.Error(err),
			)
			return nil, err
		}
		invites = append(invites, invite)
	}

	return invites, nil
}

func (r *inviteRepository) IncrementUseCount(ctx context.Context, inviteID uuid.UUID) error {
	query := `UPDATE messages.conversation_invites SET use_count = use_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, inviteID)
	if err != nil {
		r.logger.Error("Failed to increment invite use count",
			logger.String("invite_id", inviteID.String()),
			logger.Error(err),
		)
	}
	return err
}

func (r *inviteRepository) RevokeInvite(ctx context.Context, inviteID uuid.UUID) error {
	query := `UPDATE messages.conversation_invites SET status = 'revoked', revoked_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, inviteID)
	if err != nil {
		r.logger.Error("Failed to revoke invite",
			logger.String("invite_id", inviteID.String()),
			logger.Error(err),
		)
	}
	return err
}
