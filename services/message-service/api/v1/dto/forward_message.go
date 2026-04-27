package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type ForwardMessageRequest struct {
	ConversationIDs []string `json:"conversation_ids" validate:"required,min=1,max=5,dive,uuid4"`
	Comment         *string  `json:"comment,omitempty" validate:"omitempty,max=500"`
}

func NewForwardMessageRequest() *ForwardMessageRequest {
	return &ForwardMessageRequest{}
}

func (r *ForwardMessageRequest) GetValue() interface{} {
	return r
}

func (r *ForwardMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationIDs":
			switch fieldErr.Tag() {
			case "required", "min":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "conversation_ids must contain at least one target",
				})
			case "max":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "conversation_ids must contain at most 5 targets",
				})
			case "uuid4":
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "each conversation_ids entry must be a valid UUID",
				})
			}
		case "Comment":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "comment must be at most 500 characters",
				})
			}
		}
	}
	return errors, nil
}

type ForwardedTarget struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

type ForwardMessageResponse struct {
	ForwardedTo []ForwardedTarget `json:"forwarded_to"`
}
