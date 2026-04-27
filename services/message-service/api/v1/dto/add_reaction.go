package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type AddReactionRequest struct {
	ReactionType string  `json:"reaction_type" validate:"required,min=1,max=100"`
	SkinTone     *string `json:"skin_tone,omitempty" validate:"omitempty,oneof=light medium_light medium medium_dark dark"`
}

func NewAddReactionRequest() *AddReactionRequest {
	return &AddReactionRequest{}
}

func (r *AddReactionRequest) GetValue() interface{} {
	return r
}

func (r *AddReactionRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ReactionType":
			switch fieldErr.Tag() {
			case "required", "min":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "reaction_type is required",
				})
			case "max":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "reaction_type must be at most 100 characters",
				})
			}
		case "SkinTone":
			if fieldErr.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "skin_tone must be one of: light, medium_light, medium, medium_dark, dark",
				})
			}
		}
	}
	return errors, nil
}
