package domain

import (
	db "shared/pkg/database/postgres/models"
	"shared/pkg/utils"
	"time"
)

type DeviceType string

const (
	DeviceTypeAndroid DeviceType = "android"
	DeviceTypeIOS     DeviceType = "ios"
	DeviceTypeWeb     DeviceType = "web"
	DeviceTypeDesktop DeviceType = "desktop"
)

type UserDevice struct {
	ID                 string
	UserID             string
	DeviceID           string
	DeviceName         string
	DeviceType         *DeviceType
	DeviceModel        *string
	DeviceManufacturer *string
	OSName             *string
	OSVersion          *string
	AppVersion         *string
	FCMToken           *string
	APNSToken          *string
	IsCurrentDevice    bool
	IsActive           bool
	PushEnabled        bool
	LastActiveAt       *time.Time
	RegisteredAt       time.Time
}

func NewUserDevice(model db.Device) *UserDevice {
	return &UserDevice{
		ID:         model.ID,
		UserID:     model.UserID,
		DeviceID:   model.DeviceID,
		DeviceName: *utils.SafePtrString(model.DeviceName),
		DeviceType: func() *DeviceType {
			if model.DeviceType == nil {
				return nil
			}
			dt := DeviceType(*model.DeviceType)
			return &dt
		}(),
		DeviceModel:        utils.SafePtrString(model.DeviceModel),
		DeviceManufacturer: utils.SafePtrString(model.DeviceManufacturer),
		OSName:             utils.SafePtrString(model.OSName),
		OSVersion:          utils.SafePtrString(model.OSVersion),
		AppVersion:         utils.SafePtrString(model.AppVersion),
		FCMToken:           utils.SafePtrString(model.FCMToken),
		APNSToken:          utils.SafePtrString(model.APNSToken),
		IsCurrentDevice:    model.IsCurrentDevice,
		IsActive:           model.IsActive,
		PushEnabled:        model.PushEnabled,
		LastActiveAt:       utils.Ptr(model.LastActiveAt),
		RegisteredAt:       model.RegisteredAt,
	}
}

type UpdateUserDevice struct {
	AppVersion  *string
	FCMToken    *string
	APNSToken   *string
	PushEnabled *bool
	IsActive    *bool
}
