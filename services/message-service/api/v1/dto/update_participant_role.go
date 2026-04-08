package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpdateParticipantRoleRequest represents the request to update a participant's role
type UpdateParticipantRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member moderator"`
}

func NewUpdateParticipantRoleRequest() *UpdateParticipantRoleRequest {
	return &UpdateParticipantRoleRequest{}
}

func (r *UpdateParticipantRoleRequest) GetValue() interface{} {
	return r
}

func (r *UpdateParticipantRoleRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "Role" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Role is required",
				})
			} else if fieldErr.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Role must be one of: admin, member, moderator",
				})
			}
		}
	}
	return errors, nil
}
