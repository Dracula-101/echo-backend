package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// CreatePollRequest represents the request to create a poll
type CreatePollRequest struct {
	ConversationID string   `json:"conversation_id" validate:"required,uuid4"`
	Question       string   `json:"question" validate:"required,min=1,max=500"`
	Options        []string `json:"options" validate:"required,min=2,max=10,dive,min=1,max=200"`
	AllowMultiple  bool     `json:"allow_multiple"`
	ExpiresAt      *string  `json:"expires_at,omitempty"`
}

func NewCreatePollRequest() *CreatePollRequest {
	return &CreatePollRequest{}
}

func (r *CreatePollRequest) GetValue() interface{} {
	return r
}

func (r *CreatePollRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationID":
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
		case "Question":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Poll question is required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_SHORT,
					Msg:  "Poll question cannot be empty",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Poll question must be at most 500 characters",
				})
			}
		case "Options":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Poll options are required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "At least 2 poll options are required",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "At most 10 poll options are allowed",
				})
			}
		}
	}
	return errors, nil
}
