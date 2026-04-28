package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type SendOTPRequest struct {
	PhoneNumber      string `json:"phone_number" validate:"required,e164"`
	PhoneCountryCode string `json:"phone_country_code" validate:"required,iso3166_1_alpha2"`
}

func NewSendOTPRequest() *SendOTPRequest {
	return &SendOTPRequest{}
}

func (r *SendOTPRequest) GetValue() interface{} {
	return r
}

func (r *SendOTPRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "PhoneNumber":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Phone number is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "e164" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Phone number must be in E.164 format",
					Code: request.INVALID_FORMAT,
				})
			}
		case "PhoneCountryCode":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Phone country code is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "iso3166_1_alpha2" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "Phone country code must be a valid ISO 3166-1 alpha-2 code",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errs, nil
}
