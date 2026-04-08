package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// CreateContactGroupRequest represents a request to create a contact group
type CreateContactGroupRequest struct {
	Name       string   `json:"name" validate:"required,max=50"`
	ContactIDs []string `json:"contact_ids,omitempty" validate:"omitempty,dive,uuid4"`
}

func NewCreateContactGroupRequest() *CreateContactGroupRequest {
	return &CreateContactGroupRequest{}
}

func (r *CreateContactGroupRequest) GetValue() interface{} {
	return r
}

func (r *CreateContactGroupRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Name":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Group name is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Group name must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "ContactIDs":
			if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Contact IDs must be valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}

// UpdateContactGroupRequest represents a request to update a contact group
type UpdateContactGroupRequest struct {
	Name       *string  `json:"name,omitempty" validate:"omitempty,max=50"`
	ContactIDs []string `json:"contact_ids,omitempty" validate:"omitempty,dive,uuid4"`
}

func NewUpdateContactGroupRequest() *UpdateContactGroupRequest {
	return &UpdateContactGroupRequest{}
}

func (r *UpdateContactGroupRequest) GetValue() interface{} {
	return r
}

func (r *UpdateContactGroupRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Name":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Group name must be at most 50 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "ContactIDs":
			if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Contact IDs must be valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
