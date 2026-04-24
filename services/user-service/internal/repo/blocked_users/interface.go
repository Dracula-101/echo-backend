package blocked_users

import (
	"context"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

type BlockedRepository interface {
	IsBlocked(ctx context.Context, userID, targetID string) (bool, pkgErrors.AppError)
	GetBlockedUser(ctx context.Context, userID, targetID string) (*domain.BlockedUser, pkgErrors.AppError)
	GetBlockedUserIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError)
	GetBlockedUsers(ctx context.Context, userID string) ([]*domain.BlockedUser, pkgErrors.AppError)
	// BlockUser calls the users.block_user stored function and returns the new block ID.
	BlockUser(ctx context.Context, userID, targetID string, reason *string, blockType string) (string, pkgErrors.AppError)
	UnblockUser(ctx context.Context, userID, targetID string) pkgErrors.AppError
}

var _ BlockedRepository = (*blockedRepository)(nil)
