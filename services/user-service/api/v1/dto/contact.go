package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// AddContactRequest represents a request to add a contact
type AddContactRequest struct {
	ContactUserID string  `json:"contact_user_id" validate:"required,uuid4"`
	Nickname      *string `json:"nickname,omitempty" validate:"omitempty,max=50"`
}

func NewAddContactRequest() *AddContactRequest {
	return &AddContactRequest{}
}

func (r *AddContactRequest) GetValue() interface{} {
	return r
}

func (r *AddContactRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "ContactUserID":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Contact user ID is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Contact user ID must be a valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Nickname":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Nickname must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}

// UpdateContactRequest represents a request to update a contact
type UpdateContactRequest struct {
	Nickname *string `json:"nickname,omitempty" validate:"omitempty,max=50"`
	IsMuted  *bool   `json:"is_muted,omitempty"`
}

func NewUpdateContactRequest() *UpdateContactRequest {
	return &UpdateContactRequest{}
}

func (r *UpdateContactRequest) GetValue() interface{} {
	return r
}

func (r *UpdateContactRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Nickname":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Nickname must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
