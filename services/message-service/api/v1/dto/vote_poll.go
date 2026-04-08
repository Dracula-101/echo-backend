package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// VotePollRequest represents the request to vote on a poll
type VotePollRequest struct {
	OptionIndexes []int `json:"option_indexes" validate:"required,min=1,dive,min=0"`
}

func NewVotePollRequest() *VotePollRequest {
	return &VotePollRequest{}
}

func (r *VotePollRequest) GetValue() interface{} {
	return r
}

func (r *VotePollRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "OptionIndexes" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Option indexes are required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "At least one option index is required",
				})
			}
		}
	}
	return errors, nil
}
