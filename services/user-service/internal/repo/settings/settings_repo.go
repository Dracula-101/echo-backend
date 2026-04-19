package settings

import (
	"context"
	"fmt"
	"strings"

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type settingsRepository struct {
	db  database.Database
	log logger.Logger
}

func NewSettingsRepository(db database.Database, log logger.Logger) SettingsRepository {
	if db == nil {
		panic("database is required for SettingsRepository")
	}
	if log == nil {
		panic("logger is required for SettingsRepository")
	}
	return &settingsRepository{db: db, log: log}
}

func (r *settingsRepository) GetSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError) {
	s := &dbModels.UserSettings{}
	if dbErr := r.db.FindOne(ctx, s, `SELECT * FROM users.settings WHERE user_id = $1`, userID); dbErr != nil {
		if postgres.IsNotFoundError(dbErr) {
			return nil, nil
		}
		r.log.Error("failed to get settings",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get settings")
	}
	return domain.FromDBModel(s), nil
}

func (r *settingsRepository) CreateSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError) {
	query := `
		INSERT INTO users.settings (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`

	if _, dbErr := r.db.Exec(ctx, query, userID); dbErr != nil {
		r.log.Error("failed to create settings",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to create settings")
	}
	return r.GetSettings(ctx, userID)
}

func (r *settingsRepository) UpdateSettings(ctx context.Context, userID string, params UpdateSettingsParams) (*domain.Settings, pkgErrors.AppError) {
	setClauses := []string{"updated_at = NOW()"}
	args := []any{}
	argIdx := 1

	addParam := func(col string, val any) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}

	if params.ProfileVisibility != nil {
		addParam("profile_visibility", *params.ProfileVisibility)
	}
	if params.LastSeenVisibility != nil {
		addParam("last_seen_visibility", *params.LastSeenVisibility)
	}
	if params.OnlineStatusVisibility != nil {
		addParam("online_status_visibility", *params.OnlineStatusVisibility)
	}
	if params.ProfilePhotoVisibility != nil {
		addParam("profile_photo_visibility", *params.ProfilePhotoVisibility)
	}
	if params.AboutVisibility != nil {
		addParam("about_visibility", *params.AboutVisibility)
	}
	if params.ReadReceiptsEnabled != nil {
		addParam("read_receipts_enabled", *params.ReadReceiptsEnabled)
	}
	if params.TypingIndicatorsEnabled != nil {
		addParam("typing_indicators_enabled", *params.TypingIndicatorsEnabled)
	}
	if params.PushNotificationsEnabled != nil {
		addParam("push_notifications_enabled", *params.PushNotificationsEnabled)
	}
	if params.EmailNotificationsEnabled != nil {
		addParam("email_notifications_enabled", *params.EmailNotificationsEnabled)
	}
	if params.SmsNotificationsEnabled != nil {
		addParam("sms_notifications_enabled", *params.SmsNotificationsEnabled)
	}
	if params.MessageNotifications != nil {
		addParam("message_notifications", *params.MessageNotifications)
	}
	if params.GroupMessageNotifications != nil {
		addParam("group_message_notifications", *params.GroupMessageNotifications)
	}
	if params.MentionNotifications != nil {
		addParam("mention_notifications", *params.MentionNotifications)
	}
	if params.ReactionNotifications != nil {
		addParam("reaction_notifications", *params.ReactionNotifications)
	}
	if params.CallNotifications != nil {
		addParam("call_notifications", *params.CallNotifications)
	}
	if params.NotificationSound != nil {
		addParam("notification_sound", *params.NotificationSound)
	}
	if params.VibrationEnabled != nil {
		addParam("vibration_enabled", *params.VibrationEnabled)
	}
	if params.NotificationPreview != nil {
		addParam("notification_preview", *params.NotificationPreview)
	}
	if params.QuietHoursEnabled != nil {
		addParam("quiet_hours_enabled", *params.QuietHoursEnabled)
	}
	if params.QuietHoursStart != nil {
		addParam("quiet_hours_start", *params.QuietHoursStart)
	}
	if params.QuietHoursEnd != nil {
		addParam("quiet_hours_end", *params.QuietHoursEnd)
	}
	if params.EnterKeyToSend != nil {
		addParam("enter_key_to_send", *params.EnterKeyToSend)
	}
	if params.AutoDownloadPhotos != nil {
		addParam("auto_download_photos", *params.AutoDownloadPhotos)
	}
	if params.AutoDownloadVideos != nil {
		addParam("auto_download_videos", *params.AutoDownloadVideos)
	}
	if params.AutoDownloadDocuments != nil {
		addParam("auto_download_documents", *params.AutoDownloadDocuments)
	}
	if params.AutoDownloadOnWifiOnly != nil {
		addParam("auto_download_on_wifi_only", *params.AutoDownloadOnWifiOnly)
	}
	if params.CompressImages != nil {
		addParam("compress_images", *params.CompressImages)
	}
	if params.SaveToGallery != nil {
		addParam("save_to_gallery", *params.SaveToGallery)
	}
	if params.ChatBackupEnabled != nil {
		addParam("chat_backup_enabled", *params.ChatBackupEnabled)
	}
	if params.ChatBackupFrequency != nil {
		addParam("chat_backup_frequency", *params.ChatBackupFrequency)
	}
	if params.ScreenLockEnabled != nil {
		addParam("screen_lock_enabled", *params.ScreenLockEnabled)
	}
	if params.ScreenLockTimeout != nil {
		addParam("screen_lock_timeout", *params.ScreenLockTimeout)
	}
	if params.FingerprintUnlock != nil {
		addParam("fingerprint_unlock", *params.FingerprintUnlock)
	}
	if params.FaceUnlock != nil {
		addParam("face_unlock", *params.FaceUnlock)
	}
	if params.ShowSecurityNotifications != nil {
		addParam("show_security_notifications", *params.ShowSecurityNotifications)
	}
	if params.Theme != nil {
		addParam("theme", *params.Theme)
	}
	if params.FontSize != nil {
		addParam("font_size", *params.FontSize)
	}
	if params.ChatWallpaper != nil {
		addParam("chat_wallpaper", *params.ChatWallpaper)
	}
	if params.UseSystemEmoji != nil {
		addParam("use_system_emoji", *params.UseSystemEmoji)
	}
	if params.LanguageCode != nil {
		addParam("language_code", *params.LanguageCode)
	}
	if params.Timezone != nil {
		addParam("timezone", *params.Timezone)
	}
	if params.DateFormat != nil {
		addParam("date_format", *params.DateFormat)
	}
	if params.TimeFormat != nil {
		addParam("time_format", *params.TimeFormat)
	}
	if params.LowDataMode != nil {
		addParam("low_data_mode", *params.LowDataMode)
	}

	args = append(args, userID)
	query := fmt.Sprintf(`
		UPDATE users.settings SET %s
		WHERE user_id = $%d
		RETURNING *`,
		strings.Join(setClauses, ", "), argIdx,
	)

	s := &dbModels.UserSettings{}
	if dbErr := r.db.FindOneAndUpdate(ctx, s, query, args...); dbErr != nil {
		r.log.Error("failed to update settings",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to update settings")
	}
	return domain.FromDBModel(s), nil
}
