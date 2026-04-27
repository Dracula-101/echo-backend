package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// EditMessageRequest represents the request to edit a message
type EditMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=65536"`
}

func NewEditMessageRequest() *EditMessageRequest {
	return &EditMessageRequest{}
}

func (r *EditMessageRequest) GetValue() interface{} {
	return r
}

func (r *EditMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "Content" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Message content is required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_SHORT,
					Msg:  "Message content cannot be empty",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Message content must be at most 65536 characters",
				})
			}
		}
	}
	return errors, nil
}
