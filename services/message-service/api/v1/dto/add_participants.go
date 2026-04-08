package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// AddParticipantsRequest represents the request to add participants to a conversation
type AddParticipantsRequest struct {
	ParticipantIDs []string `json:"participant_ids" validate:"required,min=1,dive,uuid4"`
}

func NewAddParticipantsRequest() *AddParticipantsRequest {
	return &AddParticipantsRequest{}
}

func (r *AddParticipantsRequest) GetValue() interface{} {
	return r
}

func (r *AddParticipantsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "ParticipantIDs" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Participant IDs are required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "At least one participant ID is required",
				})
			}
		}
	}
	return errors, nil
}
