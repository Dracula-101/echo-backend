package repo

import (
	"context"
	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"

	"github.com/google/uuid"
)

type UserRepository interface {
	CheckForBlockedUser(userID uuid.UUID) (bool, pkgErrors.AppError)
}

type userRepository struct {
	db database.Database
}

func NewUserRepository(db database.Database) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CheckForBlockedUser(userID uuid.UUID) (bool, pkgErrors.AppError) {
	var isBlocked bool
	query := `SELECT EXISTS (
		SELECT 1 FROM users.blocked_users
		WHERE blocked_user_id = $1
	)`
	err := r.db.QueryRow(context.Background(), query, userID).Scan(&isBlocked)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeInternal, "Failed to check for blocked user")
	}

	return isBlocked, nil
}
