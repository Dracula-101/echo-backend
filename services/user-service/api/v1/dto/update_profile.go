package dto

import (
	"shared/pkg/utils"
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// UpdateProfileRequest represents a request to update a user profile
type UpdateProfileRequest struct {
	DisplayName       *string    `json:"display_name,omitempty" validate:"omitempty,min=1,max=100"`
	FirstName         *string    `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName          *string    `json:"last_name,omitempty" validate:"omitempty,max=50"`
	Bio               *string    `json:"bio,omitempty" validate:"omitempty,max=500"`
	BioLinks          *[]string  `json:"bio_links,omitempty" validate:"omitempty,dive,url"`
	DateOfBirth       *time.Time `json:"date_of_birth,omitempty" validate:"omitempty"`
	AvatarURL         *string    `json:"avatar_url,omitempty" validate:"omitempty,url"`
	LanguageCode      *string    `json:"language_code,omitempty" validate:"omitempty,len=2"`
	Timezone          *string    `json:"timezone,omitempty"`
	CountryCode       *string    `json:"country_code,omitempty" validate:"omitempty,len=2"`
	SocialLinks       *[]string  `json:"social_links,omitempty" validate:"omitempty,dive,url"`
	ProfileVisibility *string    `json:"profile_visibility,omitempty" validate:"omitempty,oneof=public private friends"`
}

func NewUpdateProfileRequest() *UpdateProfileRequest {
	return &UpdateProfileRequest{}
}

func (r *UpdateProfileRequest) GetValue() interface{} {
	return r
}
func (r *UpdateProfileRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail

	// validator tag errors
	for _, err := range ve {
		switch err.Field() {
		case "DisplayName":
			if err.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Display name must be at least 1 character long",
					Code: request.INVALID_FORMAT,
				})
			} else if err.Tag() == "max" {
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

		case "BioLinks":
			if err.Tag() == "url" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Each bio link must be a valid URL",
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
					Msg:  "Language code must be exactly 2 characters long",
					Code: request.INVALID_FORMAT,
				})
			}

		case "CountryCode":
			if err.Tag() == "len" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Country code must be exactly 2 characters long",
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

		case "SocialLinks":
			if err.Tag() == "url" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Each social link must be a valid URL",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}

	// custom validations
	if r.Bio != nil {
		*r.Bio = utils.SanitizeHTMLText(*r.Bio)
	}

	if r.DateOfBirth != nil {
		now := time.Now()

		if r.DateOfBirth.After(now) {
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Date of birth must be in the past",
				Code: request.INVALID_FORMAT,
			})
		}

		if r.DateOfBirth.Before(now.AddDate(-150, 0, 0)) {
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Date of birth must be within the last 150 years",
				Code: request.INVALID_FORMAT,
			})
		}

		if r.DateOfBirth.After(now.AddDate(-13, 0, 0)) {
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "User must be at least 13 years old",
				Code: request.INVALID_FORMAT,
			})
		}
	}

	if r.SocialLinks != nil {
		// max 5 social links allowed
		if len(*r.SocialLinks) > 5 {
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "A maximum of 5 social links are allowed",
				Code: request.INVALID_FORMAT,
			})
		}
	}

	if r.LanguageCode != nil && !utils.IsValidLanguageCode(*r.LanguageCode) {
		errors = append(errors, request.ValidationErrorDetail{
			Msg:  "Invalid language code",
			Code: request.INVALID_FORMAT,
		})
	}

	if r.Timezone != nil && !utils.IsValidTimezone(*r.Timezone) {
		errors = append(errors, request.ValidationErrorDetail{
			Msg:  "Invalid timezone",
			Code: request.INVALID_FORMAT,
		})
	}

	if r.CountryCode != nil && !utils.IsValidCountryCode(*r.CountryCode) {
		errors = append(errors, request.ValidationErrorDetail{
			Msg:  "Invalid country code",
			Code: request.INVALID_FORMAT,
		})
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
