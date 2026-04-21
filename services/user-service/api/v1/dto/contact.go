package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// AddContactRequest represents a request to add a contact
type AddContactRequest struct {
	IdentifierType IdentifierType `json:"identifier_type" validate:"required,oneof=username phone user_id"`
	Identifier     string         `json:"identifier" validate:"required"`
	Message        *string        `json:"message,omitempty" validate:"omitempty,max=200"`
	Source         ContactSource  `json:"source,omitempty" validate:"omitempty,oneof=search qr_code phone_import suggested"`
}

type IdentifierType string

const (
	IdentifierTypeUsername IdentifierType = "username"
	IdentifierTypePhone    IdentifierType = "phone"
	IdentifierTypeUserID   IdentifierType = "user_id"
)

type ContactSource string

const (
	ContactSourceSearch      ContactSource = "search"
	ContactSourceQRCode      ContactSource = "qr_code"
	ContactSourcePhoneImport ContactSource = "phone_import"
	ContactSourceSuggested   ContactSource = "suggested"
)

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
		case "IdentifierType":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Identifier type must be one of: username, phone, user_id",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Identifier":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Identifier is required",
					Code: request.REQUIRED_FIELD,
				})
			}
		case "Message":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Message must be at most 200 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Source":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Source must be one of: search, qr_code, phone_import, suggested",
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
