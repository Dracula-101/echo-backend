package domain

import (
	"time"

	dbModels "shared/pkg/database/postgres/models"
)

type Settings struct {
	UserID                    string              `json:"user_id"`
	ProfileVisibility         ProfileVisibility   `json:"profile_visibility"`
	LastSeenVisibility        ProfileVisibility   `json:"last_seen_visibility"`
	OnlineStatusVisibility    ProfileVisibility   `json:"online_status_visibility"`
	ProfilePhotoVisibility    ProfileVisibility   `json:"profile_photo_visibility"`
	AboutVisibility           ProfileVisibility   `json:"about_visibility"`
	ReadReceiptsEnabled       bool                `json:"read_receipts_enabled"`
	TypingIndicatorsEnabled   bool                `json:"typing_indicators_enabled"`
	PushNotificationsEnabled  bool                `json:"push_notifications_enabled"`
	EmailNotificationsEnabled bool                `json:"email_notifications_enabled"`
	SmsNotificationsEnabled   bool                `json:"sms_notifications_enabled"`
	MessageNotifications      bool                `json:"message_notifications"`
	GroupMessageNotifications bool                `json:"group_message_notifications"`
	MentionNotifications      bool                `json:"mention_notifications"`
	ReactionNotifications     bool                `json:"reaction_notifications"`
	CallNotifications         bool                `json:"call_notifications"`
	NotificationSound         NotificationSound   `json:"notification_sound"`
	VibrationEnabled          bool                `json:"vibration_enabled"`
	NotificationPreview       NotificationPreview `json:"notification_preview"`
	QuietHoursEnabled         bool                `json:"quiet_hours_enabled"`
	QuietHoursStart           *time.Time          `json:"quiet_hours_start"`
	QuietHoursEnd             *time.Time          `json:"quiet_hours_end"`
	EnterKeyToSend            bool                `json:"enter_key_to_send"`
	AutoDownloadPhotos        bool                `json:"auto_download_photos"`
	AutoDownloadVideos        bool                `json:"auto_download_videos"`
	AutoDownloadDocuments     bool                `json:"auto_download_documents"`
	AutoDownloadOnWifiOnly    bool                `json:"auto_download_on_wifi_only"`
	CompressImages            bool                `json:"compress_images"`
	SaveToGallery             bool                `json:"save_to_gallery"`
	ChatBackupEnabled         bool                `json:"chat_backup_enabled"`
	ChatBackupFrequency       BackupFrequency     `json:"chat_backup_frequency"`
	ScreenLockEnabled         bool                `json:"screen_lock_enabled"`
	ScreenLockTimeout         int                 `json:"screen_lock_timeout"`
	FingerprintUnlock         bool                `json:"fingerprint_unlock"`
	FaceUnlock                bool                `json:"face_unlock"`
	ShowSecurityNotifications bool                `json:"show_security_notifications"`
	Theme                     Theme               `json:"theme"`
	FontSize                  FontSize            `json:"font_size"`
	ChatWallpaper             *string             `json:"chat_wallpaper"`
	UseSystemEmoji            bool                `json:"use_system_emoji"`
	LanguageCode              string              `json:"language_code"`
	Timezone                  *string             `json:"timezone"`
	DateFormat                DateFormat          `json:"date_format"`
	TimeFormat                TimeFormat          `json:"time_format"`
	LowDataMode               bool                `json:"low_data_mode"`
}

func FromDBModel(s *dbModels.UserSettings) *Settings {
	return &Settings{
		UserID:                    s.UserID,
		ProfileVisibility:         ProfileVisibility(s.ProfileVisibility),
		LastSeenVisibility:        ProfileVisibility(s.LastSeenVisibility),
		OnlineStatusVisibility:    ProfileVisibility(s.OnlineStatusVisibility),
		ProfilePhotoVisibility:    ProfileVisibility(s.ProfilePhotoVisibility),
		AboutVisibility:           ProfileVisibility(s.AboutVisibility),
		ReadReceiptsEnabled:       s.ReadReceiptsEnabled,
		TypingIndicatorsEnabled:   s.TypingIndicatorsEnabled,
		PushNotificationsEnabled:  s.PushNotificationsEnabled,
		EmailNotificationsEnabled: s.EmailNotificationsEnabled,
		SmsNotificationsEnabled:   s.SMSNotificationsEnabled,
		MessageNotifications:      s.MessageNotifications,
		GroupMessageNotifications: s.GroupMessageNotifications,
		MentionNotifications:      s.MentionNotifications,
		ReactionNotifications:     s.ReactionNotifications,
		CallNotifications:         s.CallNotifications,
		NotificationSound:         NotificationSound(s.NotificationSound),
		VibrationEnabled:          s.VibrationEnabled,
		NotificationPreview:       NotificationPreview(s.NotificationPreview),
		QuietHoursEnabled:         s.QuietHoursEnabled,
		QuietHoursStart:           s.QuietHoursStart,
		QuietHoursEnd:             s.QuietHoursEnd,
		EnterKeyToSend:            s.EnterKeyToSend,
		AutoDownloadPhotos:        s.AutoDownloadPhotos,
		AutoDownloadVideos:        s.AutoDownloadVideos,
		AutoDownloadDocuments:     s.AutoDownloadDocuments,
		AutoDownloadOnWifiOnly:    s.AutoDownloadOnWifiOnly,
		CompressImages:            s.CompressImages,
		SaveToGallery:             s.SaveToGallery,
		ChatBackupEnabled:         s.ChatBackupEnabled,
		ChatBackupFrequency:       BackupFrequency(s.ChatBackupFrequency),
		ScreenLockEnabled:         s.ScreenLockEnabled,
		ScreenLockTimeout:         s.ScreenLockTimeout,
		FingerprintUnlock:         s.FingerprintUnlock,
		FaceUnlock:                s.FaceUnlock,
		ShowSecurityNotifications: s.ShowSecurityNotifications,
		Theme:                     Theme(s.Theme),
		FontSize:                  FontSize(s.FontSize),
		ChatWallpaper:             s.ChatWallpaper,
		UseSystemEmoji:            s.UseSystemEmoji,
		LanguageCode:              s.LanguageCode,
		Timezone:                  s.Timezone,
		DateFormat:                DateFormat(s.DateFormat),
		TimeFormat:                TimeFormat(s.TimeFormat),
		LowDataMode:               s.LowDataMode,
	}
}
