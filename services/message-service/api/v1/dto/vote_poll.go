package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type VotePollRequest struct {
	OptionIDs []string `json:"option_ids" validate:"required,min=1,dive,uuid4"`
}

func NewVotePollRequest() *VotePollRequest {
	return &VotePollRequest{}
}

func (r *VotePollRequest) GetValue() interface{} {
	return r
}

func (r *VotePollRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		if fieldErr.Field() == "OptionIDs" {
			switch fieldErr.Tag() {
			case "required", "min":
				errors = append(errors, request.ValidationErrorDetail{Code: request.REQUIRED_FIELD, Msg: "option_ids is required"})
			case "uuid4":
				errors = append(errors, request.ValidationErrorDetail{Code: request.INVALID_FORMAT, Msg: "each option_ids entry must be a valid UUID"})
			}
		}
	}
	return errors, nil
}
