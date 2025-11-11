package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Profile struct {
	ID                 string            `db:"id" json:"id" pk:"true"`
	UserID             string            `db:"user_id" json:"user_id"`
	Username           string            `db:"username" json:"username"`
	DisplayName        *string           `db:"display_name" json:"display_name,omitempty"`
	FirstName          *string           `db:"first_name" json:"first_name,omitempty"`
	LastName           *string           `db:"last_name" json:"last_name,omitempty"`
	MiddleName         *string           `db:"middle_name" json:"middle_name,omitempty"`
	Bio                *string           `db:"bio" json:"bio,omitempty"`
	BioLinks           *json.RawMessage  `db:"bio_links" json:"bio_links,omitempty"`
	AvatarURL          *string           `db:"avatar_url" json:"avatar_url,omitempty"`
	AvatarThumbnailURL *string           `db:"avatar_thumbnail_url" json:"avatar_thumbnail_url,omitempty"`
	CoverImageURL      *string           `db:"cover_image_url" json:"cover_image_url,omitempty"`
	DateOfBirth        *time.Time        `db:"date_of_birth" json:"date_of_birth,omitempty"`
	Gender             *string           `db:"gender" json:"gender,omitempty"`
	Pronouns           *string           `db:"pronouns" json:"pronouns,omitempty"`
	LanguageCode       string            `db:"language_code" json:"language_code"`
	Timezone           *string           `db:"timezone" json:"timezone,omitempty"`
	CountryCode        *string           `db:"country_code" json:"country_code,omitempty"`
	City               *string           `db:"city" json:"city,omitempty"`
	PhoneVisible       bool              `db:"phone_visible" json:"phone_visible"`
	EmailVisible       bool              `db:"email_visible" json:"email_visible"`
	OnlineStatus       OnlineStatus      `db:"online_status" json:"online_status"`
	LastSeenAt         *time.Time        `db:"last_seen_at" json:"last_seen_at,omitempty"`
	ProfileVisibility  ProfileVisibility `db:"profile_visibility" json:"profile_visibility"`
	SearchVisibility   bool              `db:"search_visibility" json:"search_visibility"`
	IsVerified         bool              `db:"is_verified" json:"is_verified"`
	WebsiteURL         *string           `db:"website_url" json:"website_url,omitempty"`
	SocialLinks        *json.RawMessage  `db:"social_links" json:"social_links,omitempty"`
	Interests          pq.StringArray    `db:"interests" json:"interests,omitempty"`
	CreatedAt          *time.Time        `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt          *time.Time        `db:"updated_at" json:"updated_at,omitempty"`
	DeactivatedAt      *time.Time        `db:"deactivated_at" json:"deactivated_at,omitempty"`
	Metadata           *json.RawMessage  `db:"metadata" json:"metadata,omitempty"`
}

func (p *Profile) TableName() string {
	return "users.profiles"
}

func (p *Profile) PrimaryKey() interface{} {
	return p.ID
}

type Contact struct {
	ID                  string           `db:"id" json:"id" pk:"true"`
	UserID              string           `db:"user_id" json:"user_id"`
	ContactUserID       string           `db:"contact_user_id" json:"contact_user_id"`
	RelationshipType    RelationshipType `db:"relationship_type" json:"relationship_type"`
	Status              ContactStatus    `db:"status" json:"status"`
	Nickname            *string          `db:"nickname" json:"nickname,omitempty"`
	Notes               *string          `db:"notes" json:"notes,omitempty"`
	IsFavorite          bool             `db:"is_favorite" json:"is_favorite"`
	IsPinned            bool             `db:"is_pinned" json:"is_pinned"`
	IsArchived          bool             `db:"is_archived" json:"is_archived"`
	IsMuted             bool             `db:"is_muted" json:"is_muted"`
	MutedUntil          *time.Time       `db:"muted_until" json:"muted_until,omitempty"`
	CustomNotifications *json.RawMessage `db:"custom_notifications" json:"custom_notifications,omitempty"`
	ContactSource       *string          `db:"contact_source" json:"contact_source,omitempty"`
	ContactGroups       pq.StringArray   `db:"contact_groups" json:"contact_groups,omitempty"`
	LastInteractionAt   *time.Time       `db:"last_interaction_at" json:"last_interaction_at,omitempty"`
	InteractionCount    int              `db:"interaction_count" json:"interaction_count"`
	CreatedAt           *time.Time       `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt           *time.Time       `db:"updated_at" json:"updated_at,omitempty"`
	AcceptedAt          *time.Time       `db:"accepted_at" json:"accepted_at,omitempty"`
	BlockedAt           *time.Time       `db:"blocked_at" json:"blocked_at,omitempty"`
	BlockReason         *string          `db:"block_reason" json:"block_reason,omitempty"`
}

func (c *Contact) TableName() string {
	return "users.contacts"
}

func (c *Contact) PrimaryKey() interface{} {
	return c.ID
}

type ContactGroup struct {
	ID          string    `db:"id" json:"id" pk:"true"`
	UserID      string    `db:"user_id" json:"user_id"`
	GroupName   string    `db:"group_name" json:"group_name"`
	GroupColor  *string   `db:"group_color" json:"group_color,omitempty"`
	GroupIcon   *string   `db:"group_icon" json:"group_icon,omitempty"`
	Description *string   `db:"description" json:"description,omitempty"`
	MemberCount int       `db:"member_count" json:"member_count"`
	IsDefault   bool      `db:"is_default" json:"is_default"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

func (c *ContactGroup) TableName() string {
	return "users.contact_groups"
}

func (c *ContactGroup) PrimaryKey() interface{} {
	return c.ID
}

type UserSettings struct {
	ID                        string              `db:"id" json:"id" pk:"true"`
	UserID                    string              `db:"user_id" json:"user_id"`
	ProfileVisibility         ProfileVisibility   `db:"profile_visibility" json:"profile_visibility"`
	LastSeenVisibility        ProfileVisibility   `db:"last_seen_visibility" json:"last_seen_visibility"`
	OnlineStatusVisibility    ProfileVisibility   `db:"online_status_visibility" json:"online_status_visibility"`
	ProfilePhotoVisibility    ProfileVisibility   `db:"profile_photo_visibility" json:"profile_photo_visibility"`
	AboutVisibility           ProfileVisibility   `db:"about_visibility" json:"about_visibility"`
	ReadReceiptsEnabled       bool                `db:"read_receipts_enabled" json:"read_receipts_enabled"`
	TypingIndicatorsEnabled   bool                `db:"typing_indicators_enabled" json:"typing_indicators_enabled"`
	PushNotificationsEnabled  bool                `db:"push_notifications_enabled" json:"push_notifications_enabled"`
	EmailNotificationsEnabled bool                `db:"email_notifications_enabled" json:"email_notifications_enabled"`
	SMSNotificationsEnabled   bool                `db:"sms_notifications_enabled" json:"sms_notifications_enabled"`
	MessageNotifications      bool                `db:"message_notifications" json:"message_notifications"`
	GroupMessageNotifications bool                `db:"group_message_notifications" json:"group_message_notifications"`
	MentionNotifications      bool                `db:"mention_notifications" json:"mention_notifications"`
	ReactionNotifications     bool                `db:"reaction_notifications" json:"reaction_notifications"`
	CallNotifications         bool                `db:"call_notifications" json:"call_notifications"`
	NotificationSound         NotificationSound   `db:"notification_sound" json:"notification_sound"`
	VibrationEnabled          bool                `db:"vibration_enabled" json:"vibration_enabled"`
	NotificationPreview       NotificationPreview `db:"notification_preview" json:"notification_preview"`
	QuietHoursEnabled         bool                `db:"quiet_hours_enabled" json:"quiet_hours_enabled"`
	QuietHoursStart           *time.Time          `db:"quiet_hours_start" json:"quiet_hours_start,omitempty"`
	QuietHoursEnd             *time.Time          `db:"quiet_hours_end" json:"quiet_hours_end,omitempty"`
	EnterKeyToSend            bool                `db:"enter_key_to_send" json:"enter_key_to_send"`
	AutoDownloadPhotos        bool                `db:"auto_download_photos" json:"auto_download_photos"`
	AutoDownloadVideos        bool                `db:"auto_download_videos" json:"auto_download_videos"`
	AutoDownloadDocuments     bool                `db:"auto_download_documents" json:"auto_download_documents"`
	AutoDownloadOnWifiOnly    bool                `db:"auto_download_on_wifi_only" json:"auto_download_on_wifi_only"`
	CompressImages            bool                `db:"compress_images" json:"compress_images"`
	SaveToGallery             bool                `db:"save_to_gallery" json:"save_to_gallery"`
	ChatBackupEnabled         bool                `db:"chat_backup_enabled" json:"chat_backup_enabled"`
	ChatBackupFrequency       BackupFrequency     `db:"chat_backup_frequency" json:"chat_backup_frequency"`
	ScreenLockEnabled         bool                `db:"screen_lock_enabled" json:"screen_lock_enabled"`
	ScreenLockTimeout         int                 `db:"screen_lock_timeout" json:"screen_lock_timeout"`
	FingerprintUnlock         bool                `db:"fingerprint_unlock" json:"fingerprint_unlock"`
	FaceUnlock                bool                `db:"face_unlock" json:"face_unlock"`
	ShowSecurityNotifications bool                `db:"show_security_notifications" json:"show_security_notifications"`
	Theme                     Theme               `db:"theme" json:"theme"`
	FontSize                  FontSize            `db:"font_size" json:"font_size"`
	ChatWallpaper             *string             `db:"chat_wallpaper" json:"chat_wallpaper,omitempty"`
	UseSystemEmoji            bool                `db:"use_system_emoji" json:"use_system_emoji"`
	LanguageCode              string              `db:"language_code" json:"language_code"`
	Timezone                  *string             `db:"timezone" json:"timezone,omitempty"`
	DateFormat                DateFormat          `db:"date_format" json:"date_format"`
	TimeFormat                TimeFormat          `db:"time_format" json:"time_format"`
	LowDataMode               bool                `db:"low_data_mode" json:"low_data_mode"`
	CreatedAt                 time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time           `db:"updated_at" json:"updated_at"`
}

func (u *UserSettings) TableName() string {
	return "users.settings"
}

func (u *UserSettings) PrimaryKey() interface{} {
	return u.ID
}

type BlockedUser struct {
	ID            string          `db:"id" json:"id" pk:"true"`
	UserID        string          `db:"user_id" json:"user_id"`
	BlockedUserID string          `db:"blocked_user_id" json:"blocked_user_id"`
	BlockReason   *string         `db:"block_reason" json:"block_reason,omitempty"`
	BlockedAt     time.Time       `db:"blocked_at" json:"blocked_at"`
	UnblockedAt   *time.Time      `db:"unblocked_at" json:"unblocked_at,omitempty"`
	BlockType     BlockType       `db:"block_type" json:"block_type"`
	Metadata      json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (b *BlockedUser) TableName() string {
	return "users.blocked_users"
}

func (b *BlockedUser) PrimaryKey() interface{} {
	return b.ID
}

type PrivacyOverride struct {
	ID                      string    `db:"id" json:"id" pk:"true"`
	UserID                  string    `db:"user_id" json:"user_id"`
	TargetUserID            string    `db:"target_user_id" json:"target_user_id"`
	LastSeenVisible         *bool     `db:"last_seen_visible" json:"last_seen_visible,omitempty"`
	OnlineStatusVisible     *bool     `db:"online_status_visible" json:"online_status_visible,omitempty"`
	ProfilePhotoVisible     *bool     `db:"profile_photo_visible" json:"profile_photo_visible,omitempty"`
	AboutVisible            *bool     `db:"about_visible" json:"about_visible,omitempty"`
	ReadReceiptsEnabled     *bool     `db:"read_receipts_enabled" json:"read_receipts_enabled,omitempty"`
	TypingIndicatorsEnabled *bool     `db:"typing_indicators_enabled" json:"typing_indicators_enabled,omitempty"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time `db:"updated_at" json:"updated_at"`
}

func (p *PrivacyOverride) TableName() string {
	return "users.privacy_overrides"
}

func (p *PrivacyOverride) PrimaryKey() interface{} {
	return p.ID
}

type StatusHistory struct {
	ID              string        `db:"id" json:"id" pk:"true"`
	UserID          string        `db:"user_id" json:"user_id"`
	StatusText      *string       `db:"status_text" json:"status_text,omitempty"`
	StatusEmoji     *string       `db:"status_emoji" json:"status_emoji,omitempty"`
	MediaURL        *string       `db:"media_url" json:"media_url,omitempty"`
	MediaType       *string       `db:"media_type" json:"media_type,omitempty"`
	BackgroundColor *string       `db:"background_color" json:"background_color,omitempty"`
	ViewsCount      int           `db:"views_count" json:"views_count"`
	Privacy         StatusPrivacy `db:"privacy" json:"privacy"`
	ExpiresAt       *time.Time    `db:"expires_at" json:"expires_at,omitempty"`
	CreatedAt       time.Time     `db:"created_at" json:"created_at"`
	DeletedAt       *time.Time    `db:"deleted_at" json:"deleted_at,omitempty"`
}

func (s *StatusHistory) TableName() string {
	return "users.status_history"
}

func (s *StatusHistory) PrimaryKey() interface{} {
	return s.ID
}

type StatusView struct {
	ID           string    `db:"id" json:"id" pk:"true"`
	StatusID     string    `db:"status_id" json:"status_id"`
	ViewerUserID string    `db:"viewer_user_id" json:"viewer_user_id"`
	ViewedAt     time.Time `db:"viewed_at" json:"viewed_at"`
	ViewDuration *int      `db:"view_duration" json:"view_duration,omitempty"`
}

func (s *StatusView) TableName() string {
	return "users.status_views"
}

func (s *StatusView) PrimaryKey() interface{} {
	return s.ID
}

type ActivityLog struct {
	ID               string          `db:"id" json:"id" pk:"true"`
	UserID           string          `db:"user_id" json:"user_id"`
	ActivityType     string          `db:"activity_type" json:"activity_type"`
	ActivityCategory *string         `db:"activity_category" json:"activity_category,omitempty"`
	Description      *string         `db:"description" json:"description,omitempty"`
	OldValue         json.RawMessage `db:"old_value" json:"old_value,omitempty"`
	NewValue         json.RawMessage `db:"new_value" json:"new_value,omitempty"`
	IPAddress        *string         `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent        *string         `db:"user_agent" json:"user_agent,omitempty"`
	DeviceID         *string         `db:"device_id" json:"device_id,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
}

func (a *ActivityLog) TableName() string {
	return "users.activity_log"
}

func (a *ActivityLog) PrimaryKey() interface{} {
	return a.ID
}

type Preference struct {
	ID              string          `db:"id" json:"id" pk:"true"`
	UserID          string          `db:"user_id" json:"user_id"`
	PreferenceKey   string          `db:"preference_key" json:"preference_key"`
	PreferenceValue json.RawMessage `db:"preference_value" json:"preference_value"`
	Category        *string         `db:"category" json:"category,omitempty"`
	IsSystem        bool            `db:"is_system" json:"is_system"`
	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
}

func (p *Preference) TableName() string {
	return "users.preferences"
}

func (p *Preference) PrimaryKey() interface{} {
	return p.ID
}

type Device struct {
	ID                 string          `db:"id" json:"id" pk:"true"`
	UserID             string          `db:"user_id" json:"user_id"`
	DeviceID           string          `db:"device_id" json:"device_id"`
	DeviceName         *string         `db:"device_name" json:"device_name,omitempty"`
	DeviceType         *string         `db:"device_type" json:"device_type,omitempty"`
	DeviceModel        *string         `db:"device_model" json:"device_model,omitempty"`
	DeviceManufacturer *string         `db:"device_manufacturer" json:"device_manufacturer,omitempty"`
	OSName             *string         `db:"os_name" json:"os_name,omitempty"`
	OSVersion          *string         `db:"os_version" json:"os_version,omitempty"`
	AppVersion         *string         `db:"app_version" json:"app_version,omitempty"`
	IsCurrentDevice    bool            `db:"is_current_device" json:"is_current_device"`
	IsActive           bool            `db:"is_active" json:"is_active"`
	LastActiveAt       time.Time       `db:"last_active_at" json:"last_active_at"`
	RegisteredAt       time.Time       `db:"registered_at" json:"registered_at"`
	FCMToken           *string         `db:"fcm_token" json:"fcm_token,omitempty"`
	APNSToken          *string         `db:"apns_token" json:"apns_token,omitempty"`
	PushEnabled        bool            `db:"push_enabled" json:"push_enabled"`
	Metadata           json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (d *Device) TableName() string {
	return "users.devices"
}

func (d *Device) PrimaryKey() interface{} {
	return d.ID
}

type Achievement struct {
	ID                     string     `db:"id" json:"id" pk:"true"`
	UserID                 string     `db:"user_id" json:"user_id"`
	AchievementType        string     `db:"achievement_type" json:"achievement_type"`
	AchievementName        *string    `db:"achievement_name" json:"achievement_name,omitempty"`
	AchievementDescription *string    `db:"achievement_description" json:"achievement_description,omitempty"`
	AchievementIcon        *string    `db:"achievement_icon" json:"achievement_icon,omitempty"`
	AchievementRarity      *string    `db:"achievement_rarity" json:"achievement_rarity,omitempty"`
	Progress               int        `db:"progress" json:"progress"`
	ProgressTotal          *int       `db:"progress_total" json:"progress_total,omitempty"`
	IsUnlocked             bool       `db:"is_unlocked" json:"is_unlocked"`
	UnlockedAt             *time.Time `db:"unlocked_at" json:"unlocked_at,omitempty"`
	DisplayOnProfile       bool       `db:"display_on_profile" json:"display_on_profile"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
}

func (a *Achievement) TableName() string {
	return "users.achievements"
}

func (a *Achievement) PrimaryKey() interface{} {
	return a.ID
}

type UserReport struct {
	ID             string             `db:"id" json:"id" pk:"true"`
	ReporterUserID string             `db:"reporter_user_id" json:"reporter_user_id"`
	ReportedUserID string             `db:"reported_user_id" json:"reported_user_id"`
	ReportType     string             `db:"report_type" json:"report_type"`
	ReportCategory *string            `db:"report_category" json:"report_category,omitempty"`
	Description    *string            `db:"description" json:"description,omitempty"`
	EvidenceURLs   pq.StringArray     `db:"evidence_urls" json:"evidence_urls,omitempty"`
	Status         UserReportStatus   `db:"status" json:"status"`
	Priority       UserReportPriority `db:"priority" json:"priority"`
	AssignedTo     *string            `db:"assigned_to" json:"assigned_to,omitempty"`
	Resolution     *string            `db:"resolution" json:"resolution,omitempty"`
	ResolvedAt     *time.Time         `db:"resolved_at" json:"resolved_at,omitempty"`
	CreatedAt      time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at" json:"updated_at"`
}

func (u *UserReport) TableName() string {
	return "users.reports"
}

func (u *UserReport) PrimaryKey() interface{} {
	return u.ID
}
