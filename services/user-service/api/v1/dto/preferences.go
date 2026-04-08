package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpdatePreferencesRequest represents a request to update user preferences
type UpdatePreferencesRequest struct {
	ReadReceipts     *bool   `json:"read_receipts,omitempty"`
	TypingIndicators *bool   `json:"typing_indicators,omitempty"`
	OnlineStatus     *bool   `json:"online_status,omitempty"`
	LastSeen         *string `json:"last_seen,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
	ProfilePhoto     *string `json:"profile_photo,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
	About            *string `json:"about,omitempty" validate:"omitempty,oneof=everyone contacts nobody"`
}

func NewUpdatePreferencesRequest() *UpdatePreferencesRequest {
	return &UpdatePreferencesRequest{}
}

func (r *UpdatePreferencesRequest) GetValue() interface{} {
	return r
}

func (r *UpdatePreferencesRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "LastSeen":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Last seen visibility must be one of: everyone, contacts, nobody",
					Code: request.INVALID_FORMAT,
				})
			}
		case "ProfilePhoto":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Profile photo visibility must be one of: everyone, contacts, nobody",
					Code: request.INVALID_FORMAT,
				})
			}
		case "About":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "About visibility must be one of: everyone, contacts, nobody",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
