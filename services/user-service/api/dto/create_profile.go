package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type CreateProfileRequest struct {
	DisplayName  string `json:"display_name" validate:"required,max=50"`
	FirstName    string `json:"first_name" validate:"max=30"`
	LastName     string `json:"last_name" validate:"max=30"`
	Bio          string `json:"bio" validate:"max=160"`
	AvatarURL    string `json:"avatar_url" validate:"omitempty,url"`
	LanguageCode string `json:"language_code" validate:"omitempty,len=2"`
	Timezone     string `json:"timezone" validate:"omitempty"`
	CountryCode  string `json:"country_code" validate:"omitempty,len=2"`
}

func NewCreateProfileRequest() *CreateProfileRequest {
	return &CreateProfileRequest{}
}

func (cpr *CreateProfileRequest) GetValue() interface{} {
	return cpr
}

func (cpr *CreateProfileRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "DisplayName":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Display name is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Display name must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "FirstName":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "First name must be at most 30 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "LastName":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Last name must be at most 30 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Bio":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Bio must be at most 160 characters long",
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

type CreateProfileResponse struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Bio          string `json:"bio"`
	AvatarURL    string `json:"avatar_url"`
	LanguageCode string `json:"language_code"`
	Timezone     string `json:"timezone"`
	CountryCode  string `json:"country_code"`
	IsVerified   bool   `json:"is_verified"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}
