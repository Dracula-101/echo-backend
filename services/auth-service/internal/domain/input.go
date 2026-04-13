package domain

import (
	"shared/server/request"
	"time"
)

type RegisterUserInput struct {
	Email            string
	Password         string
	PhoneNumber      *string
	PhoneCountryCode *string
	IPAddress        string
	Country          string
	AppVersion       string
	UserAgent        string
	AcceptTerms      bool
}

type LoginInput struct {
	Email        string
	Password     string
	DeviceInfo   request.DeviceInfo
	LocationInfo *request.IpAddressInfo
}

type FailedLoginAttemptInput struct {
	UserID        string
	Device        request.DeviceInfo
	Location      *request.IpAddressInfo
	UserAgent     string
	Reason        string
	LoginMethod   string
	IsNewDevice   bool
	IsNewLocation bool
}

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
	IsNewLocation   bool
	FCMToken        string
	APNSToken       string
	SessionType     SessionType
	ExpiresAt       time.Time
	Metadata        map[string]interface{}
}

type UpdateSession struct {
	FCMToken    *string
	APNSToken   *string
	RevokedAt   *time.Time
	PushEnabled *bool
}
