package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// CreateInviteRequest represents the request to create a conversation invite
type CreateInviteRequest struct {
	MaxUses   int     `json:"max_uses" validate:"omitempty,min=1,max=1000"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func NewCreateInviteRequest() *CreateInviteRequest {
	return &CreateInviteRequest{
		MaxUses: 1,
	}
}

func (r *CreateInviteRequest) GetValue() interface{} {
	return r
}

func (r *CreateInviteRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "MaxUses" {
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  "Max uses must be between 1 and 1000",
			})
		}
	}
	return errors, nil
}
