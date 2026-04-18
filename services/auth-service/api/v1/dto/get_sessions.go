package dto

import (
	"auth-service/internal/domain"
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// GetSessionsRequest represents the payload required to list active sessions.
type GetSessionsRequest struct {
	Limit  int `json:"limit" validate:"omitempty,min=1,max=100"`
	Offset int `json:"offset" validate:"omitempty,min=0"`
}

// NewGetSessionsRequest constructs a get sessions request with default values.
func NewGetSessionsRequest() *GetSessionsRequest {
	return &GetSessionsRequest{
		Limit:  20,
		Offset: 0,
	}
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

// GetSessionsResponse communicates the list of active sessions.
type GetSessionsResponse struct {
	Sessions []domain.Session `json:"sessions"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
	Total    int              `json:"total"`
}

// NewGetSessionsResponse builds a get sessions response.
func NewGetSessionsResponse(sessions []domain.Session, limit, offset, total int) *GetSessionsResponse {
	return &GetSessionsResponse{
		Sessions: sessions,
		Limit:    limit,
		Offset:   offset,
		Total:    total,
	}
}
