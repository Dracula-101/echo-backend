package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// RegisterRequest represents the payload required to create a new account.
type RegisterRequest struct {
	Email            string  `json:"email" validate:"required,email"`
	Password         string  `json:"password" validate:"required,min=8,max=128"`
	PhoneNumber      *string `json:"phone_number,omitempty" validate:"omitempty,e164"`
	PhoneCountryCode *string `json:"phone_country_code,omitempty"`
	AcceptTerms      bool    `json:"accept_terms" validate:"required"`
}

func NewRegisterRequest() *RegisterRequest {
	return &RegisterRequest{}
}

func (r *RegisterRequest) GetValue() interface{} {
	return r
}

func (r *RegisterRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var msgs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Email":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Email is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "email" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Email must be a valid email address",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Password":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Password is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" || err.Tag() == "max" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Password must be between 8 and 128 characters",
					Code: request.TOO_SHORT,
				})
			}
		case "PhoneNumber":
			if err.Tag() == "e164" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Phone number must be in valid E.164 format",
					Code: request.INVALID_FORMAT,
				})
			}
		case "AcceptTerms":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "You must accept the terms and conditions",
					Code: request.REQUIRED_FIELD,
				})
			}
		}
	}
	return msgs, nil
}

// RegisterResponse communicates the outcome of the registration flow.
type RegisterResponse struct {
	UserID                string    `json:"user_id"`
	Email                 string    `json:"email"`
	EmailVerificationSent bool      `json:"email_verification_sent"`
	Message               string    `json:"message"`
	RequiresVerification  bool      `json:"requires_verification"`
	AccessToken           string    `json:"access_token,omitempty"`
	RefreshToken          string    `json:"refresh_token,omitempty"`
	ExpiresAt             time.Time `json:"expires_at,omitempty"`
}

func NewRegisterResponse(userID, email string, emailVerificationSent bool, accessToken, refreshToken string, expires_at time.Time) *RegisterResponse {
	return &RegisterResponse{
		UserID:                userID,
		Email:                 email,
		EmailVerificationSent: emailVerificationSent,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		ExpiresAt:             expires_at,
	}
}
