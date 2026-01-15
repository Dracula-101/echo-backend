package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// TypingIndicatorRequest represents the request to set typing indicator
type TypingIndicatorRequest struct {
	ConversationID string `json:"conversation_id" validate:"required,uuid4"`
	IsTyping       bool   `json:"is_typing"`
}

func NewTypingIndicatorRequest() *TypingIndicatorRequest {
	return &TypingIndicatorRequest{}
}

func (r *TypingIndicatorRequest) GetValue() interface{} {
	return r
}

func (r *TypingIndicatorRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "ConversationID" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Conversation ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Conversation ID must be a valid UUID",
				})
			}
		}
	}
	return errors, nil
}
