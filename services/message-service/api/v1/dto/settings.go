package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpdateSettingsRequest represents the request to update conversation settings
type UpdateSettingsRequest struct {
	DisappearingMessagesEnabled  *bool `json:"disappearing_messages_enabled,omitempty"`
	DisappearingMessagesDuration *int  `json:"disappearing_messages_duration,omitempty" validate:"omitempty,min=60"`
	ReadReceiptsEnabled          *bool `json:"read_receipts_enabled,omitempty"`
	TypingIndicatorsEnabled      *bool `json:"typing_indicators_enabled,omitempty"`
	LinkPreviewsEnabled          *bool `json:"link_previews_enabled,omitempty"`
	AutoDownloadMedia            *bool `json:"auto_download_media,omitempty"`
}

func NewUpdateSettingsRequest() *UpdateSettingsRequest {
	return &UpdateSettingsRequest{}
}

func (r *UpdateSettingsRequest) GetValue() interface{} {
	return r
}

func (r *UpdateSettingsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "DisappearingMessagesDuration" && fieldErr.Tag() == "min" {
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  "Disappearing messages duration must be at least 60 seconds",
			})
		}
	}
	return errors, nil
}
