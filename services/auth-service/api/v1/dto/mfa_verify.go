package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// MFAVerifyRequest represents the payload required to verify an MFA code.
type MFAVerifyRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Code   string `json:"code" validate:"required,len=6"`
}

// NewMFAVerifyRequest constructs an MFA verify request with zero values.
func NewMFAVerifyRequest() *MFAVerifyRequest {
	return &MFAVerifyRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *MFAVerifyRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *MFAVerifyRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "UserID":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "User ID is required",
				Code: request.REQUIRED_FIELD,
			})
		case "Code":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "MFA code is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "len" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "MFA code must be exactly 6 characters",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errs, nil
}

// MFAVerifyResponse communicates the outcome of the MFA verification.
type MFAVerifyResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// NewMFAVerifyResponse builds an MFA verify response.
func NewMFAVerifyResponse(valid bool, message string) *MFAVerifyResponse {
	return &MFAVerifyResponse{
		Valid:   valid,
		Message: message,
	}
}
