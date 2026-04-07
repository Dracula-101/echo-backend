package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ResendVerificationRequest represents the payload required to resend a verification email.
type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// NewResendVerificationRequest constructs a resend verification request with zero values.
func NewResendVerificationRequest() *ResendVerificationRequest {
	return &ResendVerificationRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *ResendVerificationRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *ResendVerificationRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Email":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Email is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "email" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Email must be a valid email address",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errs, nil
}

// ResendVerificationResponse communicates the outcome of the resend verification flow.
type ResendVerificationResponse struct {
	Message string `json:"message"`
}

// NewResendVerificationResponse builds a resend verification response.
func NewResendVerificationResponse(message string) *ResendVerificationResponse {
	return &ResendVerificationResponse{
		Message: message,
	}
}
