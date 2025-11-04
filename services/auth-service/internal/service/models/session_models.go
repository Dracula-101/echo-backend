package models

import "shared/server/request"

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
	SessionType     string
	Metadata        map[string]interface{}
}

type CreateSessionOutput struct {
	SessionId         string
	SessionToken      string
	DeviceFingerprint string
}
