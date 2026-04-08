package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// AddReactionRequest represents the request to add a reaction to a message
type AddReactionRequest struct {
	Emoji string `json:"emoji" validate:"required,min=1,max=50"`
}

func NewAddReactionRequest() *AddReactionRequest {
	return &AddReactionRequest{}
}

func (r *AddReactionRequest) GetValue() interface{} {
	return r
}

func (r *AddReactionRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "Emoji" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Emoji is required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_SHORT,
					Msg:  "Emoji cannot be empty",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Emoji must be at most 50 characters",
				})
			}
		}
	}
	return errors, nil
}
