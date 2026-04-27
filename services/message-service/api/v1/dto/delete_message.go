package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type DeleteMessageRequest struct {
	DeleteFor string `json:"delete_for,omitempty" validate:"omitempty,oneof=me everyone"`
}

func NewDeleteMessageRequest() *DeleteMessageRequest {
	return &DeleteMessageRequest{}
}

func (r *DeleteMessageRequest) GetValue() interface{} {
	return r
}

func (r *DeleteMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		if fieldErr.Field() == "DeleteFor" && fieldErr.Tag() == "oneof" {
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  "delete_for must be 'me' or 'everyone'",
			})
		}
	}
	return errors, nil
}
