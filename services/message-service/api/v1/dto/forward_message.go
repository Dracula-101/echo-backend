package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ForwardMessageRequest represents the request to forward a message to another conversation
type ForwardMessageRequest struct {
	TargetConversationID string `json:"target_conversation_id" validate:"required,uuid4"`
}

func NewForwardMessageRequest() *ForwardMessageRequest {
	return &ForwardMessageRequest{}
}

func (r *ForwardMessageRequest) GetValue() interface{} {
	return r
}

func (r *ForwardMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "TargetConversationID" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Target conversation ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Target conversation ID must be a valid UUID",
				})
			}
		}
	}
	return errors, nil
}
