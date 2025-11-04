package models

import "shared/server/request"

type CreateLoginHistoryInput struct {
	DeviceFingerprint string
	DeviceInfo        request.DeviceInfo
	IPInfo            request.IpAddressInfo
	UserID            string
	SessionID         *string
	LoginMethod       *string
	Status            *string
	FailureReason     *string
	UserAgent         *string
	IsNewDevice       *bool
	IsNewLocation     *bool
}
