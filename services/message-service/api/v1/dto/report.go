package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// ReportMessageRequest represents the request to report a message
type ReportMessageRequest struct {
	ReportType  string `json:"report_type" validate:"required,oneof=spam harassment hate_speech inappropriate misinformation other"`
	Description string `json:"description" validate:"omitempty,max=1000"`
}

func NewReportMessageRequest() *ReportMessageRequest {
	return &ReportMessageRequest{}
}

func (r *ReportMessageRequest) GetValue() interface{} {
	return r
}

func (r *ReportMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ReportType":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Report type is required",
				})
			} else if fieldErr.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Report type must be one of: spam, harassment, hate_speech, inappropriate, misinformation, other",
				})
			}
		case "Description":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Description must be at most 1000 characters",
				})
			}
		}
	}
	return errors, nil
}
