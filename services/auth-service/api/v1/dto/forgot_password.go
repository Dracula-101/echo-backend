package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ForgotPasswordRequest represents the payload required to initiate a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// NewForgotPasswordRequest constructs a forgot password request with zero values.
func NewForgotPasswordRequest() *ForgotPasswordRequest {
	return &ForgotPasswordRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *ForgotPasswordRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *ForgotPasswordRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
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

// ForgotPasswordResponse communicates the outcome of the forgot password flow.
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// NewForgotPasswordResponse builds a forgot password response.
func NewForgotPasswordResponse(message string) *ForgotPasswordResponse {
	return &ForgotPasswordResponse{
		Message: message,
	}
}
