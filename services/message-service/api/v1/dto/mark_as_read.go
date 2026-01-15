package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// MarkAsReadRequest represents the request to mark a message as read
type MarkAsReadRequest struct {
	MessageID string `json:"message_id" validate:"required,uuid4"`
}

func NewMarkAsReadRequest() *MarkAsReadRequest {
	return &MarkAsReadRequest{}
}

func (r *MarkAsReadRequest) GetValue() interface{} {
	return r
}

func (r *MarkAsReadRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "MessageID" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Message ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Message ID must be a valid UUID",
				})
			}
		}
	}
	return errors, nil
}
