package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type UpdateSettingsRequest struct {
	// Privacy
	ProfileVisibility      *string `json:"profile_visibility,omitempty" validate:"omitempty,oneof=public friends private"`
	LastSeenVisibility     *string `json:"last_seen_visibility,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
	OnlineStatusVisibility *string `json:"online_status_visibility,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
	ProfilePhotoVisibility *string `json:"profile_photo_visibility,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
	AboutVisibility        *string `json:"about_visibility,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
	ReadReceiptsEnabled    *bool   `json:"read_receipts_enabled,omitempty"`
	TypingIndicatorsEnabled *bool  `json:"typing_indicators_enabled,omitempty"`

	// Notifications
	PushNotificationsEnabled  *bool   `json:"push_notifications_enabled,omitempty"`
	EmailNotificationsEnabled *bool   `json:"email_notifications_enabled,omitempty"`
	SmsNotificationsEnabled   *bool   `json:"sms_notifications_enabled,omitempty"`
	MessageNotifications      *bool   `json:"message_notifications,omitempty"`
	GroupMessageNotifications *bool   `json:"group_message_notifications,omitempty"`
	MentionNotifications      *bool   `json:"mention_notifications,omitempty"`
	ReactionNotifications     *bool   `json:"reaction_notifications,omitempty"`
	CallNotifications         *bool   `json:"call_notifications,omitempty"`
	NotificationSound         *string `json:"notification_sound,omitempty" validate:"omitempty,oneof=default silent vibrate custom_1 custom_2 custom_3 custom_4 custom_5"`
	VibrationEnabled          *bool   `json:"vibration_enabled,omitempty"`
	NotificationPreview       *string `json:"notification_preview,omitempty" validate:"omitempty,oneof=full name_only none"`
	QuietHoursEnabled         *bool   `json:"quiet_hours_enabled,omitempty"`
	QuietHoursStart           *string `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd             *string `json:"quiet_hours_end,omitempty"`

	// Chat
	EnterKeyToSend         *bool `json:"enter_key_to_send,omitempty"`
	AutoDownloadPhotos     *bool `json:"auto_download_photos,omitempty"`
	AutoDownloadVideos     *bool `json:"auto_download_videos,omitempty"`
	AutoDownloadDocuments  *bool `json:"auto_download_documents,omitempty"`
	AutoDownloadOnWifiOnly *bool `json:"auto_download_on_wifi_only,omitempty"`
	CompressImages         *bool `json:"compress_images,omitempty"`
	SaveToGallery          *bool `json:"save_to_gallery,omitempty"`

	// Display
	Theme    *string `json:"theme,omitempty" validate:"omitempty,oneof=system light dark"`
	FontSize *string `json:"font_size,omitempty" validate:"omitempty,oneof=small medium large extra_large"`

	// Language
	LanguageCode *string `json:"language_code,omitempty"`
}

func NewUpdateSettingsRequest() *UpdateSettingsRequest {
	return &UpdateSettingsRequest{}
}

func (r *UpdateSettingsRequest) GetValue() interface{} {
	return r
}

func (r *UpdateSettingsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		var msg string
		switch err.Field() {
		case "ProfileVisibility":
			msg = "profile_visibility must be one of: public, friends, private"
		case "LastSeenVisibility":
			msg = "last_seen_visibility must be one of: everyone, contacts, nobody"
		case "OnlineStatusVisibility":
			msg = "online_status_visibility must be one of: everyone, contacts, nobody"
		case "ProfilePhotoVisibility":
			msg = "profile_photo_visibility must be one of: everyone, contacts, nobody"
		case "AboutVisibility":
			msg = "about_visibility must be one of: everyone, contacts, nobody"
		case "NotificationSound":
			msg = "notification_sound must be one of: default, silent, vibrate, custom_1..custom_5"
		case "NotificationPreview":
			msg = "notification_preview must be one of: full, name_only, none"
		case "Theme":
			msg = "theme must be one of: system, light, dark"
		case "FontSize":
			msg = "font_size must be one of: small, medium, large, extra_large"
		default:
			msg = err.Error()
		}
		errors = append(errors, request.ValidationErrorDetail{
			Msg:  msg,
			Code: request.INVALID_FORMAT,
		})
	}
	return errors, nil
}
