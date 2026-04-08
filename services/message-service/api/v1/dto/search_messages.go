package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// SearchMessagesRequest represents the request to search messages
type SearchMessagesRequest struct {
	Query          string  `json:"query" validate:"required,min=1,max=500"`
	ConversationID *string `json:"conversation_id,omitempty" validate:"omitempty,uuid4"`
	Limit          int     `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset         int     `json:"offset,omitempty" validate:"omitempty,min=0"`
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
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Search query is required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_SHORT,
					Msg:  "Search query cannot be empty",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Search query must be at most 500 characters",
				})
			}
		case "ConversationID":
			if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Conversation ID must be a valid UUID",
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
					Msg:  "Limit must be at most 100",
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
