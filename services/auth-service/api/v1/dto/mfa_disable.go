package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// MFADisableRequest represents the payload required to disable MFA for a user.
type MFADisableRequest struct {
	UserID   string `json:"user_id" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// NewMFADisableRequest constructs an MFA disable request with zero values.
func NewMFADisableRequest() *MFADisableRequest {
	return &MFADisableRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *MFADisableRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *MFADisableRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "UserID":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "User ID is required",
				Code: request.REQUIRED_FIELD,
			})
		case "Password":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Password is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Password must be at least 8 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errs, nil
}

// MFADisableResponse communicates the outcome of disabling MFA.
type MFADisableResponse struct {
	Message string `json:"message"`
}

// NewMFADisableResponse builds an MFA disable response.
func NewMFADisableResponse(message string) *MFADisableResponse {
	return &MFADisableResponse{
		Message: message,
	}
}
