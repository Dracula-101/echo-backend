package domain

import (
	"time"
)

type Settings struct {
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
