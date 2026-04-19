package privacy_overrides

import (
	"context"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

type UpsertPrivacyOverrideParams struct {
	LastSeenVisible         *bool
	OnlineStatusVisible     *bool
	ProfilePhotoVisible     *bool
	AboutVisible            *bool
	ReadReceiptsEnabled     *bool
	TypingIndicatorsEnabled *bool
}

type PrivacyOverrideRepository interface {
	GetPrivacyOverride(ctx context.Context, userID, targetUserID string) (*domain.PrivacyOverride, pkgErrors.AppError)
	GetPrivacyOverrides(ctx context.Context, userID string) ([]*domain.PrivacyOverride, pkgErrors.AppError)
	UpsertPrivacyOverride(ctx context.Context, userID, targetUserID string, params UpsertPrivacyOverrideParams) (*domain.PrivacyOverride, pkgErrors.AppError)
	DeletePrivacyOverride(ctx context.Context, userID, targetUserID string) pkgErrors.AppError
}

var _ PrivacyOverrideRepository = (*privacyOverrideRepository)(nil)
