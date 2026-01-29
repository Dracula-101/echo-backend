package domain

import (
	db "shared/pkg/database/postgres/models"
	"shared/pkg/utils"
)

type UserDevice struct {
	UserID             string
	DeviceID           string
	DeviceName         string
	DeviceType         string
	DeviceModel        string
	DeviceManufacturer string
	OSName             string
	OSVersion          string
	AppVersion         string
	IsActive           bool
	FCMToken           *string
	APNSToken          *string
	PushEnabled        bool
}

func NewUserDevice(model db.Device) *UserDevice {
	return &UserDevice{
		UserID:             model.UserID,
		DeviceID:           model.DeviceID,
		DeviceName:         *utils.SafePtrString(model.DeviceName),
		DeviceType:         *utils.SafePtrString(model.DeviceType),
		DeviceModel:        *utils.SafePtrString(model.DeviceModel),
		DeviceManufacturer: *utils.SafePtrString(model.DeviceManufacturer),
		OSName:             *utils.SafePtrString(model.OSName),
		OSVersion:          *utils.SafePtrString(model.OSVersion),
		AppVersion:         *utils.SafePtrString(model.AppVersion),
		IsActive:           model.IsActive,
		FCMToken:           model.FCMToken,
		APNSToken:          model.APNSToken,
		PushEnabled:        model.PushEnabled,
	}
}

type UpdateUserDevice struct {
	AppVersion  *string
	FCMToken    *string
	APNSToken   *string
	PushEnabled *bool
	IsActive    *bool
}
