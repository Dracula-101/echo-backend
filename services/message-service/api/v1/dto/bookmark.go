package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type CreateBookmarkRequest struct {
	CollectionName *string  `json:"collection_name,omitempty" validate:"omitempty,max=255"`
	Notes          *string  `json:"notes,omitempty" validate:"omitempty,max=500"`
	Tags           []string `json:"tags,omitempty" validate:"omitempty,max=10,dive,max=50"`
}

func NewCreateBookmarkRequest() *CreateBookmarkRequest {
	return &CreateBookmarkRequest{}
}

func (r *CreateBookmarkRequest) GetValue() interface{} {
	return r
}

func (r *CreateBookmarkRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "CollectionName":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "collection_name must be at most 255 characters"})
			}
		case "Notes":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "notes must be at most 500 characters"})
			}
		case "Tags":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "tags must contain at most 10 entries (each ≤ 50 chars)"})
			}
		}
	}
	return errors, nil
}
