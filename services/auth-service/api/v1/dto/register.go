package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// RegisterRequest represents the payload required to create a new account.
type RegisterRequest struct {
	Email            string  `json:"email" validate:"required,email,max=255"`
	Password         string  `json:"password" validate:"required,min=8,max=128"`
	PhoneNumber      *string `json:"phone_number,omitempty" validate:"omitempty,e164"`
	PhoneCountryCode *string `json:"phone_country_code,omitempty" validate:"len=2,required_with=PhoneNumber"`
	AcceptTerms      bool    `json:"accept_terms" validate:"required,eq=true"`
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
			} else if err.Tag() == "max" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Email must not exceed 255 characters",
					Code: request.TOO_LONG,
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
		case "PhoneCountryCode":
			if err.Tag() == "len" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Phone country code must be exactly 2 characters",
					Code: request.INVALID_FORMAT,
				})
			} else if err.Tag() == "required_with" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Phone country code is required when phone number is provided",
					Code: request.REQUIRED_FIELD,
				})
			}
		case "AcceptTerms":
			if err.Tag() == "required" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "Accepting terms and conditions is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "eq" {
				msgs = append(msgs, request.ValidationErrorDetail{
					Msg:  "You must accept the terms and conditions",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return msgs, nil
}

// RegisterResponse communicates the outcome of the registration flow.
type RegisterResponse struct {
	UserID                  string  `json:"user_id"`
	Email                   string  `json:"email"`
	EmailVerified           bool    `json:"email_verified"`
	PhoneVerified           bool    `json:"phone_verified"`
	AccountStatus           string  `json:"account_status"`
	NextStep                string  `json:"next_step"`
	VerificationEmailSentTo *string `json:"verification_email_sent_to,omitempty"` // masked email if sent, null if not sent
	Message                 string  `json:"message"`
}

func NewRegisterResponse(userID, email string, emailVerified, phoneVerified bool, accountStatus, nextStep string, verificationEmailSentTo string, message string) *RegisterResponse {
	return &RegisterResponse{
		UserID:                  userID,
		Email:                   email,
		EmailVerified:           emailVerified,
		PhoneVerified:           phoneVerified,
		AccountStatus:           accountStatus,
		NextStep:                nextStep,
		VerificationEmailSentTo: &verificationEmailSentTo,
		Message:                 message,
	}
}
