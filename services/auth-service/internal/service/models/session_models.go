package models

import (
	dbModels "shared/pkg/database/postgres/models"
	"shared/server/request"
)

type CreateSessionInput struct {
	UserID          string
	RefreshToken    string
	Device          request.DeviceInfo
	Browser         request.BrowserInfo
	UserAgent       string
	IP              request.IpAddressInfo
	Latitude        float64
	Longitude       float64
	IsMobile        bool
	IsTrustedDevice bool
	FCMToken        string
	APNSToken       string
	SessionType     dbModels.SessionType
	Metadata        map[string]interface{}
}

type CreateSessionOutput struct {
	SessionId         string
	SessionToken      string
	DeviceFingerprint string
}
