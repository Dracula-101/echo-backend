package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// SetStatusRequest represents a request to set a user status
type SetStatusRequest struct {
	Content    string  `json:"content" validate:"required,max=500"`
	MediaURL   *string `json:"media_url,omitempty" validate:"omitempty,url"`
	MediaType  *string `json:"media_type,omitempty" validate:"omitempty,oneof=image video"`
	Visibility *string `json:"visibility,omitempty" validate:"omitempty,oneof=public contacts private"`
	ExpiresIn  *int    `json:"expires_in,omitempty" validate:"omitempty,min=1,max=86400"`
}

func NewSetStatusRequest() *SetStatusRequest {
	return &SetStatusRequest{}
}

func (r *SetStatusRequest) GetValue() interface{} {
	return r
}

func (r *SetStatusRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "Content":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Status content is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Status content must be at most 500 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		case "MediaURL":
			if err.Tag() == "url" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Media URL must be a valid URL",
					Code: request.INVALID_FORMAT,
				})
			}
		case "MediaType":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Media type must be one of: image, video",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Visibility":
			if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Visibility must be one of: public, contacts, private",
					Code: request.INVALID_FORMAT,
				})
			}
		case "ExpiresIn":
			if err.Tag() == "min" || err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Expires in must be between 1 and 86400 seconds",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
