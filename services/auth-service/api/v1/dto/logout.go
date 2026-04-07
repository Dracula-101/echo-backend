package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// LogoutRequest represents the payload required to log out a user session.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	SessionID    string `json:"session_id" validate:"required"`
}

// NewLogoutRequest constructs a logout request with zero values.
func NewLogoutRequest() *LogoutRequest {
	return &LogoutRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *LogoutRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *LogoutRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "RefreshToken":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Refresh token is required",
				Code: request.REQUIRED_FIELD,
			})
		case "SessionID":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Session ID is required",
				Code: request.REQUIRED_FIELD,
			})
		}
	}
	return errs, nil
}

// LogoutResponse communicates the outcome of the logout flow.
type LogoutResponse struct {
	Message string `json:"message"`
}

// NewLogoutResponse builds a logout response.
func NewLogoutResponse(message string) *LogoutResponse {
	return &LogoutResponse{
		Message: message,
	}
}
