package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type CreateProfileRequest struct {
	UserID            string `json:"user_id" validate:"required,uuid4"`
	DisplayName       string `json:"display_name" validate:"required,max=50"`
	Username          string `json:"username,omitempty"`
	FirstName         string `json:"first_name" validate:"max=30"`
	LastName          string `json:"last_name" validate:"max=30"`
	Bio               string `json:"bio" validate:"max=160"`
	LanguageCode      string `json:"language_code" validate:"omitempty,len=2"`
	ProfileVisibility string `json:"profile_visibility" validate:"required,oneof=public private friends"`
	Searchable        *bool  `json:"searchable" validate:"required"`
	PushEnabled       *bool  `json:"push_enabled" validate:"required"`
}

func NewCreateProfileRequest() *CreateProfileRequest {
	return &CreateProfileRequest{}
}

func (r *CreateProfileRequest) GetValue() interface{} {
	return r
}

func addValidationError(
	errors *[]request.ValidationErrorDetail,
	msg string,
	code string,
) {
	*errors = append(*errors, request.ValidationErrorDetail{
		Msg:  msg,
		Code: code,
	})
}

func (r *CreateProfileRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail

	for _, err := range ve {
		field := err.Field()
		tag := err.Tag()

		switch field {
		case "UserID":
			switch tag {
			case "required":
				addValidationError(&errors, "User ID is required", request.REQUIRED_FIELD)
			case "uuid4":
				addValidationError(&errors, "User ID must be a valid UUIDv4", request.INVALID_FORMAT)
			}

		case "DisplayName":
			switch tag {
			case "required":
				addValidationError(&errors, "Display name is required", request.REQUIRED_FIELD)
			case "max":
				addValidationError(&errors, "Display name must be at most 50 characters long", request.INVALID_FORMAT)
			}

		case "FirstName":
			if tag == "max" {
				addValidationError(&errors, "First name must be at most 30 characters long", request.INVALID_FORMAT)
			}

		case "LastName":
			if tag == "max" {
				addValidationError(&errors, "Last name must be at most 30 characters long", request.INVALID_FORMAT)
			}

		case "Bio":
			if tag == "max" {
				addValidationError(&errors, "Bio must be at most 160 characters long", request.INVALID_FORMAT)
			}

		case "LanguageCode":
			if tag == "len" {
				addValidationError(&errors, "Language code must be 2 characters long", request.INVALID_FORMAT)
			}

		case "ProfileVisibility":
			switch tag {
			case "required":
				addValidationError(&errors, "Profile visibility is required", request.REQUIRED_FIELD)
			case "oneof":
				addValidationError(&errors, "Profile visibility must be one of 'public', 'private', or 'friends'", request.INVALID_FORMAT)
			}

		case "Searchable":
			if tag == "required" {
				addValidationError(&errors, "Searchable field is required", request.REQUIRED_FIELD)
			}

		case "PushEnabled":
			if tag == "required" {
				addValidationError(&errors, "Push enabled field is required", request.REQUIRED_FIELD)
			}
		}
	}

	if r.LanguageCode != "" && len(r.LanguageCode) != 2 {
		addValidationError(&errors, "Language code must be 2 characters long", request.INVALID_FORMAT)
	}

	return errors, nil
}

type CreateProfileResponse struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	DisplayName        string `json:"display_name"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Bio                string `json:"bio,omitempty"`
	AvatarURL          string `json:"avatar_url,omitempty"`
	AvatarThumbnailURL string `json:"avatar_thumbnail_url,omitempty"`
	LanguageCode       string `json:"language_code,omitempty"`
	Timezone           string `json:"timezone"`
	CountryCode        string `json:"country_code"`
	IsVerified         bool   `json:"is_verified"`
	PhoneVerified      bool   `json:"phone_verified"`
	EmailVerified      bool   `json:"email_verified"`
	TwoFactorEnabled   bool   `json:"two_factor_enabled"`
	AccountStatus      string `json:"account_status"`
}
