package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpsertDraftRequest represents the request to save/update a draft
type UpsertDraftRequest struct {
	Content          string  `json:"content" validate:"required,min=1"`
	ReplyToMessageID *string `json:"reply_to_message_id,omitempty" validate:"omitempty,uuid4"`
}

func NewUpsertDraftRequest() *UpsertDraftRequest {
	return &UpsertDraftRequest{}
}

func (r *UpsertDraftRequest) GetValue() interface{} {
	return r
}

func (r *UpsertDraftRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "Content":
			if fieldErr.Tag() == "required" || fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Draft content is required",
				})
			}
		case "ReplyToMessageID":
			if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Reply to message ID must be a valid UUID",
				})
			}
		}
	}
	return errors, nil
}
