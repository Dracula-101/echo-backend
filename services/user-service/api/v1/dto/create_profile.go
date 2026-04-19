package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type CreateProfileRequest struct {
	UserID            string `json:"user_id" validate:"required,uuid4"`
	DisplayName       string `json:"display_name" validate:"required,max=50"`
	FirstName         string `json:"first_name" validate:"max=30"`
	LastName          string `json:"last_name" validate:"max=30"`
	Bio               string `json:"bio" validate:"max=160"`
	LanguageCode      string `json:"language_code" validate:"omitempty,len=2"`
	Timezone          string `json:"timezone" validate:"omitempty"`
	CountryCode       string `json:"country_code" validate:"omitempty,len=2"`
	ProfileVisibility string `json:"profile_visibility" validate:"omitempty,oneof=public private friends"`
	Searchable        *bool  `json:"searchable" validate:"omitempty"`
	PushEnabled       *bool  `json:"push_enabled" validate:"omitempty"`
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
		case "UserID":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "User ID is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "User ID must be a valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
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
		case "Timezone":
			if err.Tag() == "timezone" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Timezone must be a valid IANA timezone",
					Code: request.INVALID_FORMAT,
				})
			}
		case "ProfileVisibility":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Profile visibility must be one of 'public', 'private', or 'friends'",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Searchable":
			if err.Tag() == "bool" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Searchable must be a boolean value",
					Code: request.INVALID_FORMAT,
				})
			}
		case "PushEnabled":
			if err.Tag() == "bool" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Push enabled must be a boolean value",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}

type CreateProfileResponse struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	DisplayName        string `json:"display_name"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Bio                string `json:"bio"`
	AvatarURL          string `json:"avatar_url"`
	AvatarThumbnailURL string `json:"avatar_thumbnail_url"`
	LanguageCode       string `json:"language_code"`
	Timezone           string `json:"timezone"`
	CountryCode        string `json:"country_code"`
	IsVerified         bool   `json:"is_verified"`
	PhoneVerified      bool   `json:"phone_verified"`
	EmailVerified      bool   `json:"email_verified"`
	TwoFactorEnabled   bool   `json:"two_factor_enabled"`
	AccountStatus      string `json:"account_status"`
}
