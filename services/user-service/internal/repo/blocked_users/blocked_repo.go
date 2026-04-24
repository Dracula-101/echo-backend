package blocked_users

import (
	"context"
	"strings"
	"time"

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

// blockedUserRow is a local scan struct that uses plain string for block_type
// to avoid the shared BlockType enum rejecting DB values like "full"/"messages_only".
type blockedUserRow struct {
	ID            string     `db:"id"`
	UserID        string     `db:"user_id"`
	BlockedUserID string     `db:"blocked_user_id"`
	BlockReason   *string    `db:"block_reason"`
	BlockedAt     time.Time  `db:"blocked_at"`
	UnblockedAt   *time.Time `db:"unblocked_at"`
	BlockType     string     `db:"block_type"`
}

func (r *blockedUserRow) TableName() string    { return "users.blocked_users" }
func (r *blockedUserRow) PrimaryKey() interface{} { return r.ID }

func rowToDomain(row blockedUserRow) *domain.BlockedUser {
	return &domain.BlockedUser{
		ID:            row.ID,
		UserID:        row.UserID,
		BlockedUserID: row.BlockedUserID,
		BlockType:     domain.BlockType(row.BlockType),
		Reason:        row.BlockReason,
		BlockedAt:     row.BlockedAt,
		UnblockedAt:   row.UnblockedAt,
	}
}

type blockedRepository struct {
	db  database.Database
	log logger.Logger
}

func NewBlockedRepository(db database.Database, log logger.Logger) BlockedRepository {
	if db == nil {
		panic("database is required for BlockedRepository")
	}
	if log == nil {
		panic("logger is required for BlockedRepository")
	}
	return &blockedRepository{db: db, log: log}
}

func (r *blockedRepository) IsBlocked(ctx context.Context, userID, targetID string) (bool, pkgErrors.AppError) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM users.blocked_users
			WHERE user_id = $1
			AND blocked_user_id = $2
			AND unblocked_at IS NULL
		)`

	var blocked dbModels.BlockedUser
	exists, dbErr := r.db.Exists(ctx, &blocked, query, userID, targetID)
	if dbErr != nil {
		r.log.Error("failed to check block status",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return false, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to check block status")
	}
	return exists, nil
}

func (r *blockedRepository) GetBlockedUserIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError) {
	query := `
		SELECT blocked_user_id FROM users.blocked_users
		WHERE user_id = $1
		AND unblocked_at IS NULL`

	var rows []struct {
		BlockedUserID string `db:"blocked_user_id"`
	}
	if dbErr := r.db.FindMany(ctx, &rows, query, userID); dbErr != nil {
		r.log.Error("failed to get blocked user IDs",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get blocked user IDs")
	}

	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.BlockedUserID
	}
	return ids, nil
}

func (r *blockedRepository) BlockUser(ctx context.Context, userID, targetID string, reason *string, blockType string) (string, pkgErrors.AppError) {
	if blockType == "" {
		blockType = "full"
	}

	var blockID string
	row := r.db.QueryRow(ctx, `SELECT users.block_user($1::UUID, $2::UUID, $3, $4)`, userID, targetID, reason, blockType)
	if dbErr := row.Scan(&blockID); dbErr != nil {
		r.log.Error("failed to block user",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		if strings.Contains(dbErr.Error(), "already blocked") {
			return "", pkgErrors.New(userErrors.ErrCodeUserAlreadyBlocked, "user is already blocked")
		}
		return "", pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to block user")
	}
	return blockID, nil
}

func (r *blockedRepository) GetBlockedUser(ctx context.Context, userID, targetID string) (*domain.BlockedUser, pkgErrors.AppError) {
	query := `
		SELECT id, user_id, blocked_user_id, block_reason, blocked_at, unblocked_at, block_type
		FROM users.blocked_users
		WHERE user_id = $1::UUID
		AND blocked_user_id = $2::UUID
		AND unblocked_at IS NULL`

	var row blockedUserRow
	if dbErr := r.db.FindOne(ctx, &row, query, userID, targetID); dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return nil, nil
		}
		r.log.Error("failed to get blocked user",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get blocked user")
	}
	return rowToDomain(row), nil
}

func (r *blockedRepository) GetBlockedUsers(ctx context.Context, userID string) ([]*domain.BlockedUser, pkgErrors.AppError) {
	query := `
		SELECT id, user_id, blocked_user_id, block_reason, blocked_at, unblocked_at, block_type
		FROM users.blocked_users
		WHERE user_id = $1::UUID
		AND unblocked_at IS NULL
		ORDER BY blocked_at DESC`

	var rows []blockedUserRow
	if dbErr := r.db.FindMany(ctx, &rows, query, userID); dbErr != nil {
		r.log.Error("failed to get blocked users",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get blocked users")
	}

	list := make([]*domain.BlockedUser, len(rows))
	for i, row := range rows {
		list[i] = rowToDomain(row)
	}
	return list, nil
}

func (r *blockedRepository) UnblockUser(ctx context.Context, userID, targetID string) pkgErrors.AppError {
	if _, dbErr := r.db.Exec(ctx, `SELECT users.unblock_user($1::UUID, $2::UUID)`, userID, targetID); dbErr != nil {
		r.log.Error("failed to unblock user",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to unblock user")
	}
	return nil
}
