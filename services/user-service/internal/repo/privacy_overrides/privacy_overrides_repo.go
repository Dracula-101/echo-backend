package privacy_overrides

import (
	"context"

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type privacyOverrideRepository struct {
	db  database.Database
	log logger.Logger
}

func NewPrivacyOverrideRepository(db database.Database, log logger.Logger) PrivacyOverrideRepository {
	if db == nil {
		panic("database is required for PrivacyOverrideRepository")
	}
	if log == nil {
		panic("logger is required for PrivacyOverrideRepository")
	}
	return &privacyOverrideRepository{db: db, log: log}
}

func (r *privacyOverrideRepository) GetPrivacyOverride(ctx context.Context, userID, targetUserID string) (*domain.PrivacyOverride, pkgErrors.AppError) {
	o := &dbModels.PrivacyOverride{}
	if dbErr := r.db.FindOne(ctx, o, `SELECT * FROM users.privacy_overrides WHERE user_id = $1 AND target_user_id = $2`, userID, targetUserID); dbErr != nil {
		return nil, nil
	}
	return domain.NewPrivacyOverride(*o), nil
}

func (r *privacyOverrideRepository) GetPrivacyOverrides(ctx context.Context, userID string) ([]*domain.PrivacyOverride, pkgErrors.AppError) {
	query := `
		SELECT * FROM users.privacy_overrides
		WHERE user_id = $1
		ORDER BY created_at DESC`

	var rows []dbModels.PrivacyOverride
	if dbErr := r.db.FindMany(ctx, &rows, query, userID); dbErr != nil {
		r.log.Error("failed to get privacy overrides",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get privacy overrides")
	}

	list := make([]*domain.PrivacyOverride, len(rows))
	for i, row := range rows {
		list[i] = domain.NewPrivacyOverride(row)
	}
	return list, nil
}

func (r *privacyOverrideRepository) UpsertPrivacyOverride(ctx context.Context, userID, targetUserID string, params UpsertPrivacyOverrideParams) (*domain.PrivacyOverride, pkgErrors.AppError) {
	query := `
		INSERT INTO users.privacy_overrides
			(user_id, target_user_id,
			 last_seen_visible, online_status_visible, profile_photo_visible,
			 about_visible, read_receipts_enabled, typing_indicators_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, target_user_id) DO UPDATE SET
			last_seen_visible         = COALESCE($3, privacy_overrides.last_seen_visible),
			online_status_visible     = COALESCE($4, privacy_overrides.online_status_visible),
			profile_photo_visible     = COALESCE($5, privacy_overrides.profile_photo_visible),
			about_visible             = COALESCE($6, privacy_overrides.about_visible),
			read_receipts_enabled     = COALESCE($7, privacy_overrides.read_receipts_enabled),
			typing_indicators_enabled = COALESCE($8, privacy_overrides.typing_indicators_enabled),
			updated_at                = NOW()
		RETURNING *`

	o := &dbModels.PrivacyOverride{}
	if dbErr := r.db.FindOneAndUpdate(ctx, o, query,
		userID, targetUserID,
		params.LastSeenVisible, params.OnlineStatusVisible, params.ProfilePhotoVisible,
		params.AboutVisible, params.ReadReceiptsEnabled, params.TypingIndicatorsEnabled,
	); dbErr != nil {
		r.log.Error("failed to upsert privacy override",
			logger.String("user_id", userID),
			logger.String("target_user_id", targetUserID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to upsert privacy override")
	}
	return domain.NewPrivacyOverride(*o), nil
}

func (r *privacyOverrideRepository) DeletePrivacyOverride(ctx context.Context, userID, targetUserID string) pkgErrors.AppError {
	query := `
		DELETE FROM users.privacy_overrides
		WHERE user_id = $1
		AND target_user_id = $2`

	if _, err := r.db.Exec(ctx, query, userID, targetUserID); err != nil {
		r.log.Error("failed to delete privacy override",
			logger.String("user_id", userID),
			logger.String("target_user_id", targetUserID),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "failed to delete privacy override")
	}
	return nil
}
