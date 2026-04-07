package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// VerifyEmailRequest represents the payload required to verify a user's email address.
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// NewVerifyEmailRequest constructs a verify email request with zero values.
func NewVerifyEmailRequest() *VerifyEmailRequest {
	return &VerifyEmailRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *VerifyEmailRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *VerifyEmailRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Token":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Verification token is required",
				Code: request.REQUIRED_FIELD,
			})
		}
	}
	return errs, nil
}

// VerifyEmailResponse communicates the outcome of the email verification flow.
type VerifyEmailResponse struct {
	Message string `json:"message"`
}

// NewVerifyEmailResponse builds a verify email response.
func NewVerifyEmailResponse(message string) *VerifyEmailResponse {
	return &VerifyEmailResponse{
		Message: message,
	}
}
