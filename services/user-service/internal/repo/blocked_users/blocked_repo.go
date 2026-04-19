package blocked_users

import (
	"context"

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

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

func (r *blockedRepository) BlockUser(ctx context.Context, userID, targetID string, reason *string, blockType string) pkgErrors.AppError {
	model := &dbModels.BlockedUser{
		UserID:        userID,
		BlockedUserID: targetID,
		BlockReason:   reason,
		BlockType:     dbModels.BlockType(blockType),
	}
	if dbErr := r.db.Upsert(ctx, model); dbErr != nil {
		r.log.Error("failed to block user",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to block user")
	}
	return nil
}

func (r *blockedRepository) GetBlockedUser(ctx context.Context, userID, targetID string) (*domain.BlockedUser, pkgErrors.AppError) {
	query := `
		SELECT * FROM users.blocked_users
		WHERE user_id = $1
		AND blocked_user_id = $2
		AND unblocked_at IS NULL`

	var model dbModels.BlockedUser
	if dbErr := r.db.FindOne(ctx, &model, query, userID, targetID); dbErr != nil {
		return nil, nil
	}
	return domain.NewBlockedUser(model), nil
}

func (r *blockedRepository) GetBlockedUsers(ctx context.Context, userID string) ([]*domain.BlockedUser, pkgErrors.AppError) {
	query := `
		SELECT * FROM users.blocked_users
		WHERE user_id = $1
		AND unblocked_at IS NULL
		ORDER BY blocked_at DESC`

	var rows []dbModels.BlockedUser
	if dbErr := r.db.FindMany(ctx, &rows, query, userID); dbErr != nil {
		r.log.Error("failed to get blocked users",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get blocked users")
	}

	list := make([]*domain.BlockedUser, len(rows))
	for i, row := range rows {
		list[i] = domain.NewBlockedUser(row)
	}
	return list, nil
}

func (r *blockedRepository) UnblockUser(ctx context.Context, userID, targetID string) pkgErrors.AppError {
	query := `
		UPDATE users.blocked_users
		SET unblocked_at = NOW(),
		    updated_at   = NOW()
		WHERE user_id = $1
		AND blocked_user_id = $2
		AND unblocked_at IS NULL`

	if _, dbErr := r.db.Exec(ctx, query, userID, targetID); dbErr != nil {
		r.log.Error("failed to unblock user",
			logger.String("user_id", userID),
			logger.String("target_id", targetID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to unblock user")
	}
	return nil
}
