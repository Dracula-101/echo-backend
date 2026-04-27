package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type SearchMessagesRequest struct {
	Query  string `json:"q" validate:"required,min=2,max=100"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=50"`
	Offset int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	Cursor string `json:"cursor,omitempty"`
}

func NewSearchMessagesRequest() *SearchMessagesRequest {
	return &SearchMessagesRequest{
		Limit:  20,
		Offset: 0,
	}
}

func (r *SearchMessagesRequest) GetValue() interface{} {
	return r
}

func (r *SearchMessagesRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "Query":
			switch fieldErr.Tag() {
			case "required":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "q is required",
				})
			case "min":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_SHORT,
					Msg:  "q must be at least 2 characters",
				})
			case "max":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "q must be at most 100 characters",
				})
			}
		case "Limit":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Limit must be at least 1",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Limit must be at most 50",
				})
			}
		case "Offset":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Offset must be non-negative",
				})
			}
		}
	}
	return errors, nil
}
