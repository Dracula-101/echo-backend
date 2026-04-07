package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// MFAEnableRequest represents the payload required to enable MFA for a user.
type MFAEnableRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Method string `json:"method" validate:"required,oneof=totp sms email"`
}

// NewMFAEnableRequest constructs an MFA enable request with zero values.
func NewMFAEnableRequest() *MFAEnableRequest {
	return &MFAEnableRequest{}
}

// GetValue exposes the request for validation helpers.
func (r *MFAEnableRequest) GetValue() interface{} {
	return r
}

// ValidateErrors maps validator errors into transport-friendly details.
func (r *MFAEnableRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errs []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "UserID":
			errs = append(errs, request.ValidationErrorDetail{
				Msg:  "User ID is required",
				Code: request.REQUIRED_FIELD,
			})
		case "Method":
			if err.Tag() == "required" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "MFA method is required",
					Code: request.REQUIRED_FIELD,
				})
			} else if err.Tag() == "oneof" {
				errs = append(errs, request.ValidationErrorDetail{
					Msg:  "MFA method must be one of: totp, sms, email",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errs, nil
}

// MFAEnableResponse communicates the outcome of enabling MFA.
type MFAEnableResponse struct {
	Secret    string `json:"secret,omitempty"`
	QRCodeURL string `json:"qr_code_url,omitempty"`
	Message   string `json:"message"`
}

// NewMFAEnableResponse builds an MFA enable response.
func NewMFAEnableResponse(secret string, qrCodeURL string, message string) *MFAEnableResponse {
	return &MFAEnableResponse{
		Secret:    secret,
		QRCodeURL: qrCodeURL,
		Message:   message,
	}
}
