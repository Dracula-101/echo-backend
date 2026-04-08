package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// Notification Preferences
// ============================================================================

// UpdatePreferencesRequest represents the payload to update notification preferences.
type UpdatePreferencesRequest struct {
	EmailEnabled   *bool    `json:"email_enabled" validate:"omitempty"`
	PushEnabled    *bool    `json:"push_enabled" validate:"omitempty"`
	SMSEnabled     *bool    `json:"sms_enabled" validate:"omitempty"`
	MutedChannels  []string `json:"muted_channels" validate:"omitempty"`
	QuietHoursFrom *string  `json:"quiet_hours_from" validate:"omitempty"`
	QuietHoursTo   *string  `json:"quiet_hours_to" validate:"omitempty"`
}

func NewUpdatePreferencesRequest() *UpdatePreferencesRequest {
	return &UpdatePreferencesRequest{}
}

func (r *UpdatePreferencesRequest) GetValue() interface{} {
	return r
}

func (r *UpdatePreferencesRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "QuietHoursFrom", "QuietHoursTo":
			msgs = append(msgs, request.ValidationErrorDetail{
				Msg:  "Quiet hours must be in valid HH:MM format",
				Code: request.INVALID_FORMAT,
			})
		}
	}
	return msgs, nil
}

// PreferencesResponse represents the notification preferences response.
type PreferencesResponse struct {
	UserID         string   `json:"user_id"`
	EmailEnabled   bool     `json:"email_enabled"`
	PushEnabled    bool     `json:"push_enabled"`
	SMSEnabled     bool     `json:"sms_enabled"`
	MutedChannels  []string `json:"muted_channels,omitempty"`
	QuietHoursFrom *string  `json:"quiet_hours_from,omitempty"`
	QuietHoursTo   *string  `json:"quiet_hours_to,omitempty"`
}
