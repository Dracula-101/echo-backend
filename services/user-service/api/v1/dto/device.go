package dto

import (
	"time"

	"user-service/internal/domain"

	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type DeviceResponse struct {
	DeviceID           string     `json:"device_id"`
	DeviceName         string     `json:"device_name,omitempty"`
	DeviceType         *string    `json:"device_type,omitempty"`
	DeviceModel        *string    `json:"device_model,omitempty"`
	DeviceManufacturer *string    `json:"device_manufacturer,omitempty"`
	OSName             *string    `json:"os_name,omitempty"`
	OSVersion          *string    `json:"os_version,omitempty"`
	AppVersion         *string    `json:"app_version,omitempty"`
	IsCurrentDevice    bool       `json:"is_current_device"`
	LastActiveAt       *time.Time `json:"last_active_at,omitempty"`
	PushEnabled        bool       `json:"push_enabled"`
	RegisteredAt       time.Time  `json:"registered_at"`
}

func NewDeviceResponse(d *domain.UserDevice, currentDeviceID string) *DeviceResponse {
	var dt *string
	if d.DeviceType != nil {
		s := string(*d.DeviceType)
		dt = &s
	}
	return &DeviceResponse{
		DeviceID:           d.DeviceID,
		DeviceName:         d.DeviceName,
		DeviceType:         dt,
		DeviceModel:        d.DeviceModel,
		DeviceManufacturer: d.DeviceManufacturer,
		OSName:             d.OSName,
		OSVersion:          d.OSVersion,
		AppVersion:         d.AppVersion,
		IsCurrentDevice:    d.DeviceID == currentDeviceID,
		LastActiveAt:       d.LastActiveAt,
		PushEnabled:        d.PushEnabled,
		RegisteredAt:       d.RegisteredAt,
	}
}

// UpdatePushTokenRequest is the body for PUT /devices/:device_id/push-token.
type UpdatePushTokenRequest struct {
	FCMToken    *string `json:"fcm_token,omitempty"`
	APNSToken   *string `json:"apns_token,omitempty"`
	PushEnabled *bool   `json:"push_enabled,omitempty"`
}

func NewUpdatePushTokenRequest() *UpdatePushTokenRequest {
	return &UpdatePushTokenRequest{}
}

func (r *UpdatePushTokenRequest) GetValue() interface{} {
	return r
}

func (r *UpdatePushTokenRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	return nil, nil
}

// UpdateDeviceRequest (kept for internal use by other flows).
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
