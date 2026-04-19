package settings

import (
	"context"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

type UpdateSettingsParams struct {
	ProfileVisibility       *string
	LastSeenVisibility      *string
	OnlineStatusVisibility  *string
	ProfilePhotoVisibility  *string
	AboutVisibility         *string
	ReadReceiptsEnabled     *bool
	TypingIndicatorsEnabled *bool

	PushNotificationsEnabled  *bool
	EmailNotificationsEnabled *bool
	SmsNotificationsEnabled   *bool
	MessageNotifications      *bool
	GroupMessageNotifications *bool
	MentionNotifications      *bool
	ReactionNotifications     *bool
	CallNotifications         *bool
	NotificationSound         *string
	VibrationEnabled          *bool
	NotificationPreview       *string
	QuietHoursEnabled         *bool
	QuietHoursStart           *string
	QuietHoursEnd             *string

	EnterKeyToSend         *bool
	AutoDownloadPhotos     *bool
	AutoDownloadVideos     *bool
	AutoDownloadDocuments  *bool
	AutoDownloadOnWifiOnly *bool
	CompressImages         *bool
	SaveToGallery          *bool
	ChatBackupEnabled      *bool
	ChatBackupFrequency    *string

	ScreenLockEnabled         *bool
	ScreenLockTimeout         *int
	FingerprintUnlock         *bool
	FaceUnlock                *bool
	ShowSecurityNotifications *bool

	Theme          *string
	FontSize       *string
	ChatWallpaper  *string
	UseSystemEmoji *bool

	LanguageCode *string
	Timezone     *string
	DateFormat   *string
	TimeFormat   *string

	LowDataMode *bool
}

type SettingsRepository interface {
	GetSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError)
	CreateSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError)
	UpdateSettings(ctx context.Context, userID string, params UpdateSettingsParams) (*domain.Settings, pkgErrors.AppError)
}

var _ SettingsRepository = (*settingsRepository)(nil)
