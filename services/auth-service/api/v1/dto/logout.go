package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// LogoutRequest represents the payload required to log out a user session.
type LogoutRequest struct {
	RevokeAllDevices bool   `json:"revoke_all_devices" validate:"omitempty"`
	RevokeReason     string `json:"revoke_reason" validate:"omitempty,oneof=user_logout lost_device suspicious_activity"`
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
		case "RevokeAllDevices":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Revoke all devices flag is required",
				Code: request.REQUIRED_FIELD,
			})
		case "RevokeReason":
			if err.Tag() == "oneof" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Revoke reason must be one of: user_logout, lost_device, suspicious_activity",
					Code: request.PATTERN_MISMATCH,
				})
			} else if err.Tag() == "string" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Revoke reason must be a string",
					Code: request.REQUIRED_FIELD,
				})
			}
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
