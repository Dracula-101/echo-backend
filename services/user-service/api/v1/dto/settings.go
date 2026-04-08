package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpdateSettingsRequest represents a request to update user settings
type UpdateSettingsRequest struct {
	NotificationsEnabled *bool   `json:"notifications_enabled,omitempty"`
	SoundEnabled         *bool   `json:"sound_enabled,omitempty"`
	VibrationEnabled     *bool   `json:"vibration_enabled,omitempty"`
	Theme                *string `json:"theme,omitempty" validate:"omitempty,oneof=light dark system"`
	FontSize             *string `json:"font_size,omitempty" validate:"omitempty,oneof=small medium large"`
	LanguageCode         *string `json:"language_code,omitempty" validate:"omitempty,len=2"`
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
		switch err.Field() {
		case "Theme":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Theme must be one of: light, dark, system",
					Code: request.INVALID_FORMAT,
				})
			}
		case "FontSize":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Font size must be one of: small, medium, large",
					Code: request.INVALID_FORMAT,
				})
			}
		case "LanguageCode":
			if err.Tag() == "len" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Language code must be 2 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
