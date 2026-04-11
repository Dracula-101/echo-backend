package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// CreateBookmarkRequest represents the request to bookmark a message
type CreateBookmarkRequest struct {
	Notes *string `json:"notes,omitempty"`
}

func NewCreateBookmarkRequest() *CreateBookmarkRequest {
	return &CreateBookmarkRequest{}
}

func (r *CreateBookmarkRequest) GetValue() interface{} {
	return r
}

func (r *CreateBookmarkRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	return nil, nil
}
