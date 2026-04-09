package domain

import (
	db "shared/pkg/database/postgres/models"
	"shared/pkg/utils"
	"shared/server/request"
	"time"
)

type UserDevice struct {
	ID          string
	UserID      string
	DeviceInfo  request.DeviceInfo
	IsActive    bool
	FCMToken    *string
	APNSToken   *string
	PushEnabled bool
}

func NewUserDevice(model db.Device) *UserDevice {
	return &UserDevice{
		ID:     model.ID,
		UserID: model.UserID,
		DeviceInfo: request.DeviceInfo{
			Name:         utils.DerefString(model.DeviceName),
			Type:         utils.DerefString(model.DeviceType),
			OS:           utils.DerefString(model.OSName),
			OsVersion:    utils.DerefString(model.OSVersion),
			AppVersion:   utils.DerefString(model.AppVersion),
			Model:        utils.DerefString(model.DeviceModel),
			Manufacturer: utils.DerefString(model.DeviceManufacturer),
		},
		IsActive:    model.IsActive,
		FCMToken:    model.FCMToken,
		APNSToken:   model.APNSToken,
		PushEnabled: model.PushEnabled,
	}
}

type UpdateUserDevice struct {
	AppVersion      *string
	FCMToken        *string
	APNSToken       *string
	PushEnabled     *bool
	IsActive        *bool
	IsCurrentDevice *bool
	LastActiveAt    *time.Time
}
