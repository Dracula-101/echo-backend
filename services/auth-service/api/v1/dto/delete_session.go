package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// DeleteSessionRequest represents the payload required to revoke a specific session.
type DeleteSessionRequest struct {
	SessionID string `json:"session_id" validate:"required"`
}

// NewDeleteSessionRequest constructs a delete session request with zero values.
func NewDeleteSessionRequest() *DeleteSessionRequest {
	return &DeleteSessionRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *DeleteSessionRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *DeleteSessionRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "SessionID":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Session ID is required",
				Code: request.REQUIRED_FIELD,
			})
		}
	}
	return errs, nil
}

// DeleteSessionResponse communicates the outcome of the session deletion flow.
type DeleteSessionResponse struct {
	Message string `json:"message"`
}

// NewDeleteSessionResponse builds a delete session response.
func NewDeleteSessionResponse(message string) *DeleteSessionResponse {
	return &DeleteSessionResponse{
		Message: message,
	}
}
