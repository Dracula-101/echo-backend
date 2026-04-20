package repository

import (
	"context"
	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

// UserRepository handles operations exclusively on users.profiles.
type UserRepository interface {
	GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError)
	GetProfileByUserIDs(ctx context.Context, userIDs []string) (map[string]*domain.Profile, pkgErrors.AppError)
	GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, pkgErrors.AppError)
	GenerateUniqueUsername(ctx context.Context, baseUsername string) (*string, pkgErrors.AppError)
	CreateProfile(ctx context.Context, userId string, profile CreateProfileInput) (*domain.Profile, pkgErrors.AppError)
	UpdateProfile(ctx context.Context, userId string, params UpdateProfileParams) (*domain.Profile, pkgErrors.AppError)
	SearchProfilesByUsername(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError)
	SearchProfilesByFullName(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError)
	SearchProfilesByEmail(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError)
	SearchProfilesByPhone(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError)
	UsernameExists(ctx context.Context, username string) (bool, pkgErrors.AppError)
}

var _ UserRepository = (*userRepository)(nil)
