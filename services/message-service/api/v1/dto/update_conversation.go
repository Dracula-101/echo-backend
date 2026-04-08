package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpdateConversationRequest represents the request to update a conversation
type UpdateConversationRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
}

func NewUpdateConversationRequest() *UpdateConversationRequest {
	return &UpdateConversationRequest{}
}

func (r *UpdateConversationRequest) GetValue() interface{} {
	return r
}

func (r *UpdateConversationRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "Title":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Title must be at most 255 characters",
				})
			}
		case "Description":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Description must be at most 1000 characters",
				})
			}
		}
	}
	return errors, nil
}
