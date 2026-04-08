package domain

import (
	"shared/pkg/database/postgres/models"
	"time"
)

// Session represents an authentication session
type Session struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	SessionToken       string                 `json:"session_token"`
	RefreshToken       string                 `json:"refresh_token"`
	DeviceID           string                 `json:"device_id"`
	DeviceType         string                 `json:"device_type"`
	DeviceName         string                 `json:"device_name"`
	DeviceOS           string                 `json:"device_os"`
	DeviceOSVersion    string                 `json:"device_os_version"`
	DeviceModel        string                 `json:"device_model"`
	DeviceManufacturer string                 `json:"device_manufacturer"`
	IPISP              string                 `json:"ip_isp"`
	BrowserName        string                 `json:"browser_name"`
	BrowserVersion     string                 `json:"browser_version"`
	IPAddress          string                 `json:"ip_address"`
	City               string                 `json:"city"`
	Region             string                 `json:"region"`
	Country            string                 `json:"country"`
	Timezone           string                 `json:"timezone"`
	Latitude           *float64               `json:"latitude,omitempty"`
	Longitude          *float64               `json:"longitude,omitempty"`
	IsTrustedDevice    bool                   `json:"is_trusted_device"`
	IsMobile           bool                   `json:"is_mobile"`
	SessionType        models.SessionType     `json:"session_type"`
	FCMToken           string                 `json:"fcm_token"`
	APNSToken          string                 `json:"apns_token"`
	PushEnabled        bool                   `json:"push_enabled"`
	UserAgent          string                 `json:"user_agent"`
	RevokedAt          *time.Time             `json:"revoked_at,omitempty"`
	RevokedReason      *string                `json:"revoked_reason,omitempty"`
	LastActivityAt     time.Time              `json:"last_activity_at"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CreatedAt          time.Time              `json:"created_at"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Session) TimeToExpiry() time.Duration {
	return time.Until(s.ExpiresAt)
}

func (s *Session) Refreshable(threshold time.Duration) bool {
	return s.TimeToExpiry() <= threshold
}

type SessionRecord struct {
	ID           string
	SessionToken string
	AccessToken  string
	RefreshToken string
}

type SessionType string

const (
	SessionTypeWeb    SessionType = "web"
	SessionTypeMobile SessionType = "mobile"
)

type UpdateAuthSession struct {
	ID            string
	RefreshToken  *string
	FCMToken      *string
	APNSToken     *string
	RevokedAt     *time.Time
	RevokedReason *string
	PushEnabled   *bool
}
