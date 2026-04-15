package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ResetPasswordRequest represents the payload required to reset a user's password.
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required,min=8,max=128"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// NewResetPasswordRequest constructs a reset password request with zero values.
func NewResetPasswordRequest() *ResetPasswordRequest {
	return &ResetPasswordRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *ResetPasswordRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *ResetPasswordRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Token":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Reset token is required",
				Code: request.REQUIRED_FIELD,
			})
		case "NewPassword":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "New password is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" || err.Tag() == "max" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "New password must be between 8 and 128 characters",
					Code: request.TOO_SHORT,
				})
			}
		}
	}
	return errs, nil
}

// ResetPasswordResponse communicates the outcome of the password reset flow.
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// NewResetPasswordResponse builds a reset password response.
func NewResetPasswordResponse(message string) *ResetPasswordResponse {
	return &ResetPasswordResponse{
		Message: message,
	}
}
