package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// DeleteAccountRequest represents the payload required to delete a user account.
type DeleteAccountRequest struct {
	Password string `json:"password" validate:"required,min=8"`
	Reason   string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// NewDeleteAccountRequest constructs a delete account request with zero values.
func NewDeleteAccountRequest() *DeleteAccountRequest {
	return &DeleteAccountRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *DeleteAccountRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *DeleteAccountRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
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
					Msg:  "Password is required for account deletion",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "min" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Password must be at least 8 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Reason":
			if err.Tag() == "max" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Reason must not exceed 500 characters",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errs, nil
}

// DeleteAccountResponse communicates the outcome of the account deletion flow.
type DeleteAccountResponse struct {
	Message string `json:"message"`
}

// NewDeleteAccountResponse builds a delete account response.
func NewDeleteAccountResponse(message string) *DeleteAccountResponse {
	return &DeleteAccountResponse{
		Message: message,
	}
}
