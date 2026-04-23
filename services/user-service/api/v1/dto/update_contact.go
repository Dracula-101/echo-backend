package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type UpdateContactRequest struct {
	Nickname      *string   `json:"nickname,omitempty" validate:"omitempty,max=100"`
	Notes         *string   `json:"notes,omitempty" validate:"omitempty,max=500"`
	IsFavorite    *bool     `json:"is_favorite,omitempty"`
	IsPinned      *bool     `json:"is_pinned,omitempty"`
	IsArchived    *bool     `json:"is_archived,omitempty"`
	IsMuted       *bool     `json:"is_muted,omitempty"`
	MutedUntil    *string   `json:"muted_until,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00,gtfield=Now"`
	ContactGroups *[]string `json:"contact_groups,omitempty" validate:"omitempty,dive,uuid4"`
}

func NewUpdateContactRequest() *UpdateContactRequest {
	return &UpdateContactRequest{}
}

func (r *UpdateContactRequest) GetValue() interface{} {
	return r
}

func (r *UpdateContactRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail

	// validator tag errors
	for _, err := range ve {
		switch err.Field() {
		case "Nickname":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Nickname must be at most 100 characters long",
					Code: request.INVALID_FORMAT,
				})
			}

		case "Notes":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Notes must be at most 500 characters long",
					Code: request.INVALID_FORMAT,
				})
			}

		case "MutedUntil":
			if err.Tag() == "datetime" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Muted until must be a valid ISO 8601 datetime",
					Code: request.INVALID_FORMAT,
				})
			} else if err.Tag() == "gtfield" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Muted until must be in the future",
					Code: request.INVALID_FORMAT,
				})
			}

		case "ContactGroups":
			if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Each contact group ID must be a valid UUID v4",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}

	return errors, nil
}
