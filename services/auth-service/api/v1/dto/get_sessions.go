package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// GetSessionsRequest represents the payload required to list active sessions.
type GetSessionsRequest struct {
	Limit  int `json:"limit" validate:"omitempty,min=1,max=100"`
	Offset int `json:"offset" validate:"omitempty,min=0"`
}

// NewGetSessionsRequest constructs a get sessions request with zero values.
func NewGetSessionsRequest() *GetSessionsRequest {
	return &GetSessionsRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *GetSessionsRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *GetSessionsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Limit":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Limit is required",
				Code: request.REQUIRED_FIELD,
			})
		case "Offset":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "Offset is required",
				Code: request.REQUIRED_FIELD,
			})
		}
	}
	return errs, nil
}

// SessionItem represents a single session in the sessions list response.
type SessionItem struct {
	ID           string    `json:"id"`
	DeviceName   string    `json:"device_name,omitempty"`
	DeviceType   string    `json:"device_type,omitempty"`
	Browser      string    `json:"browser,omitempty"`
	OS           string    `json:"os,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	Location     string    `json:"location,omitempty"`
	IsCurrent    bool      `json:"is_current"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetSessionsResponse communicates the list of active sessions.
type GetSessionsResponse struct {
	Sessions []SessionItem `json:"sessions"`
}

// NewGetSessionsResponse builds a get sessions response.
func NewGetSessionsResponse(sessions []SessionItem) *GetSessionsResponse {
	return &GetSessionsResponse{
		Sessions: sessions,
	}
}
