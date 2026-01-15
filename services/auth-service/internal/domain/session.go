package domain

import (
	"shared/pkg/database/postgres/models"
	"time"
)

// Session represents an authentication session
type Session struct {
	ID              string
	UserID          string
	SessionToken    string
	RefreshToken    string
	DeviceID        string
	DeviceType      string
	DeviceName      string
	DeviceOS        string
	BrowserName     string
	BrowserVersion  string
	IPAddress       string
	City            string
	Region          string
	Country         string
	Timezone        string
	Latitude        *float64
	Longitude       *float64
	IsTrustedDevice bool
	IsMobile        bool
	SessionType     models.SessionType
	UserAgent       string
	LastActivityAt  time.Time
	ExpiresAt       time.Time
	CreatedAt       time.Time
	Metadata        map[string]interface{}
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
