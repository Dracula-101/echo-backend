package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ChangePasswordRequest represents the payload required to change a user's password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=8"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// NewChangePasswordRequest constructs a change password request with zero values.
func NewChangePasswordRequest() *ChangePasswordRequest {
	return &ChangePasswordRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *ChangePasswordRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *ChangePasswordRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "CurrentPassword":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Current password is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Current password must be at least 8 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
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

// ChangePasswordResponse communicates the outcome of the change password flow.
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

// NewChangePasswordResponse builds a change password response.
func NewChangePasswordResponse(message string) *ChangePasswordResponse {
	return &ChangePasswordResponse{
		Message: message,
	}
}
