package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// SubmitReportRequest represents a request to submit a report
type SubmitReportRequest struct {
	ReportedUserID string  `json:"reported_user_id" validate:"required,uuid4"`
	Reason         string  `json:"reason" validate:"required,oneof=spam harassment inappropriate other"`
	Description    *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

func NewSubmitReportRequest() *SubmitReportRequest {
	return &SubmitReportRequest{}
}

func (r *SubmitReportRequest) GetValue() interface{} {
	return r
}

func (r *SubmitReportRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "ReportedUserID":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Reported user ID is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Reported user ID must be a valid UUIDv4",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Reason":
			if err.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Report reason is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Report reason must be one of: spam, harassment, inappropriate, other",
					Code: request.INVALID_FORMAT,
				})
			}
		case "Description":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Description must be at most 1000 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
