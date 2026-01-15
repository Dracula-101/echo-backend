package model

import (
	"shared/server/request"
	"time"
)

type SessionType string

const (
	SessionTypeWeb    SessionType = "web"
	SessionTypeMobile SessionType = "mobile"
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
	SessionType     SessionType
	ExpiresAt       time.Time
	Metadata        map[string]interface{}
}

type CreateSessionOutput struct {
	SessionId    string
	SessionToken string
}

type SessionRecord struct {
	ID           string
	SessionToken string
	RefreshToken string
}
