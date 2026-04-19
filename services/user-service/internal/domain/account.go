package domain

type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusPending     AccountStatus = "pending"
	AccountStatusSuspended   AccountStatus = "suspended"
	AccountStatusLocked      AccountStatus = "locked"
	AccountStatusDeactivated AccountStatus = "deactivated"
	AccountStatusDeleted     AccountStatus = "deleted"
)

func (s AccountStatus) String() string {
	return string(s)
}

type OnlineStatus string

const (
	OnlineStatusOnline  OnlineStatus = "online"
	OnlineStatusOffline OnlineStatus = "offline"
	OnlineStatusAway    OnlineStatus = "away"
	OnlineStatusBusy    OnlineStatus = "busy"
)

func (s OnlineStatus) String() string {
	return string(s)
}

type ProfileVisibility string

const (
	ProfileVisibilityPublic  ProfileVisibility = "public"
	ProfileVisibilityPrivate ProfileVisibility = "private"
)

func (s ProfileVisibility) String() string {
	return string(s)
}

type NotificationSound string

const (
	NotificationSoundDefault NotificationSound = "default"
	NotificationSoundSilent  NotificationSound = "silent"
	NotificationSoundCustom  NotificationSound = "custom"
)

// NotificationPreview represents the notification preview setting
type NotificationPreview string

const (
	NotificationPreviewFull NotificationPreview = "full"
	NotificationPreviewName NotificationPreview = "name"
	NotificationPreviewNone NotificationPreview = "none"
)

type BackupFrequency string

const (
	BackupFrequencyDaily   BackupFrequency = "daily"
	BackupFrequencyWeekly  BackupFrequency = "weekly"
	BackupFrequencyMonthly BackupFrequency = "monthly"
	BackupFrequencyNever   BackupFrequency = "never"
)

type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeSystem Theme = "system"
)

type FontSize string

const (
	FontSizeSmall  FontSize = "small"
	FontSizeMedium FontSize = "medium"
	FontSizeLarge  FontSize = "large"
)

type DateFormat string

const (
	DateFormatDDMMYYYY DateFormat = "DD/MM/YYYY"
	DateFormatMMDDYYYY DateFormat = "MM/DD/YYYY"
	DateFormatYYYYMMDD DateFormat = "YYYY-MM-DD"
)

type TimeFormat string

const (
	TimeFormat12Hour TimeFormat = "12h"
	TimeFormat24Hour TimeFormat = "24h"
)
