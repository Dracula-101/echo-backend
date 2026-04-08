package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// UpdateProfileRequest represents a request to update a user profile
type UpdateProfileRequest struct {
	Username     *string `json:"username,omitempty" validate:"omitempty,min=3,max=30,alphanum"`
	DisplayName  *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	FirstName    *string `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName     *string `json:"last_name,omitempty" validate:"omitempty,max=50"`
	Bio          *string `json:"bio,omitempty" validate:"omitempty,max=500"`
	AvatarURL    *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
	LanguageCode *string `json:"language_code,omitempty" validate:"omitempty,len=2"`
	Timezone     *string `json:"timezone,omitempty"`
	CountryCode  *string `json:"country_code,omitempty" validate:"omitempty,len=2"`
}

func NewUpdateProfileRequest() *UpdateProfileRequest {
	return &UpdateProfileRequest{}
}

func (r *UpdateProfileRequest) GetValue() interface{} {
	return r
}

func (r *UpdateProfileRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Username":
			if err.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Username must be at least 3 characters long",
					Code: request.INVALID_FORMAT,
				})
			} else if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Username must be at most 30 characters long",
					Code: request.INVALID_FORMAT,
				})
			} else if err.Tag() == "alphanum" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Username must contain only alphanumeric characters",
					Code: request.INVALID_FORMAT,
				})
			}
		case "DisplayName":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Display name must be at most 100 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "FirstName":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "First name must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "LastName":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Last name must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Bio":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Bio must be at most 500 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "AvatarURL":
			if err.Tag() == "url" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Avatar URL must be a valid URL",
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
		case "CountryCode":
			if err.Tag() == "len" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Country code must be 2 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}

// UpdateProfileResponse represents the response for updating a user profile
type UpdateProfileResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  *string   `json:"display_name,omitempty"`
	FirstName    *string   `json:"first_name,omitempty"`
	LastName     *string   `json:"last_name,omitempty"`
	Bio          *string   `json:"bio,omitempty"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	LanguageCode string    `json:"language_code"`
	Timezone     *string   `json:"timezone,omitempty"`
	CountryCode  *string   `json:"country_code,omitempty"`
	IsVerified   bool      `json:"is_verified"`
	UpdatedAt    time.Time `json:"updated_at"`
}
