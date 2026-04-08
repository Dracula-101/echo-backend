package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// UpdateDeviceRequest represents a request to update a device
type UpdateDeviceRequest struct {
	DeviceName  *string `json:"device_name,omitempty" validate:"omitempty,max=100"`
	FCMToken    *string `json:"fcm_token,omitempty"`
	APNSToken   *string `json:"apns_token,omitempty"`
	PushEnabled *bool   `json:"push_enabled,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

func NewUpdateDeviceRequest() *UpdateDeviceRequest {
	return &UpdateDeviceRequest{}
}

func (r *UpdateDeviceRequest) GetValue() interface{} {
	return r
}

func (r *UpdateDeviceRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, err := range ve {
		switch err.Field() {
		case "DeviceName":
			if err.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Device name must be at most 100 characters long",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}
