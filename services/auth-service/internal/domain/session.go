package domain

import (
	"shared/pkg/database/postgres/models"
	"shared/pkg/utils"
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
	Latitude           *float64               `json:"latitude"`
	Longitude          *float64               `json:"longitude"`
	IsTrustedDevice    bool                   `json:"is_trusted_device"`
	IsMobile           bool                   `json:"is_mobile"`
	SessionType        models.SessionType     `json:"session_type"`
	FCMToken           string                 `json:"fcm_token"`
	APNSToken          string                 `json:"apns_token"`
	PushEnabled        bool                   `json:"push_enabled"`
	UserAgent          string                 `json:"user_agent"`
	RevokedAt          *time.Time             `json:"revoked_at"`
	RevokedReason      *string                `json:"revoked_reason"`
	LastActivityAt     time.Time              `json:"last_activity_at"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CreatedAt          time.Time              `json:"created_at"`
	Metadata           map[string]interface{} `json:"metadata"`
}

func FromSessionModel(m *models.AuthSession) Session {
	metadata := make(map[string]interface{})
	if m.Metadata != nil {
		utils.UnmarshalRawMessageSafe(*m.Metadata, &metadata)
	}
	return Session{
		ID:                 m.ID,
		UserID:             m.UserID,
		SessionToken:       m.SessionToken,
		RefreshToken:       utils.SafeString(m.RefreshToken),
		DeviceID:           utils.SafeString(m.DeviceID),
		DeviceType:         utils.SafeString(m.DeviceType),
		DeviceName:         utils.SafeString(m.DeviceName),
		DeviceOS:           utils.SafeString(m.DeviceOS),
		DeviceOSVersion:    utils.SafeString(m.DeviceOSVersion),
		DeviceModel:        utils.SafeString(m.DeviceModel),
		DeviceManufacturer: utils.SafeString(m.DeviceManufacturer),
		IPISP:              utils.SafeString(m.IPISP),
		BrowserName:        utils.SafeString(m.BrowserName),
		BrowserVersion:     utils.SafeString(m.BrowserVersion),
		IPAddress:          utils.SafeString(&m.IPAddress),
		City:               utils.SafeString(m.IPCity),
		Region:             utils.SafeString(m.IPRegion),
		Country:            utils.SafeString(m.IPCountry),
		Timezone:           utils.SafeString(m.IPTimezone),
		IsTrustedDevice:    m.IsTrustedDevice,
		IsMobile:           m.IsMobile,
		SessionType:        m.SessionType,
		FCMToken:           utils.SafeString(m.FCMToken),
		APNSToken:          utils.SafeString(m.APNSToken),
		PushEnabled:        m.PushEnabled,
		UserAgent:          utils.SafeString(m.UserAgent),
		RevokedAt:          m.RevokedAt,
		RevokedReason:      m.RevokedReason,
		LastActivityAt:     m.LastActivityAt,
		ExpiresAt:          m.ExpiresAt,
		CreatedAt:          m.CreatedAt,
		Latitude:           m.Latitude,
		Longitude:          m.Longitude,
		Metadata:           metadata,
	}
}

func (s *Session) ToSessionModel(metadata map[string]interface{}) *models.AuthSession {
	metadataBytes := utils.MarshalRawMessageSafe(metadata)
	return &models.AuthSession{
		ID:                 s.ID,
		UserID:             s.UserID,
		SessionToken:       s.SessionToken,
		RefreshToken:       utils.PtrString(s.RefreshToken),
		DeviceID:           utils.PtrString(s.DeviceID),
		DeviceType:         utils.PtrString(s.DeviceType),
		DeviceName:         utils.PtrString(s.DeviceName),
		DeviceOS:           utils.PtrString(s.DeviceOS),
		DeviceOSVersion:    utils.PtrString(s.DeviceOSVersion),
		DeviceModel:        utils.PtrString(s.DeviceModel),
		DeviceManufacturer: utils.PtrString(s.DeviceManufacturer),
		IPISP:              utils.PtrString(s.IPISP),
		BrowserName:        utils.PtrString(s.BrowserName),
		BrowserVersion:     utils.PtrString(s.BrowserVersion),
		IPAddress:          s.IPAddress,
		IPCity:             utils.PtrString(s.City),
		IPRegion:           utils.PtrString(s.Region),
		IPCountry:          utils.PtrString(s.Country),
		IPTimezone:         utils.PtrString(s.Timezone),
		IsTrustedDevice:    s.IsTrustedDevice,
		IsMobile:           s.IsMobile,
		SessionType:        s.SessionType,
		FCMToken:           utils.PtrString(s.FCMToken),
		APNSToken:          utils.PtrString(s.APNSToken),
		PushEnabled:        s.PushEnabled,
		UserAgent:          utils.PtrString(s.UserAgent),
		RevokedAt:          s.RevokedAt,
		RevokedReason:      s.RevokedReason,
		LastActivityAt:     s.LastActivityAt,
		ExpiresAt:          s.ExpiresAt,
		CreatedAt:          s.CreatedAt,
		Latitude:           s.Latitude,
		Longitude:          s.Longitude,
		Metadata:           &metadataBytes,
	}
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

type SessionData struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	DeviceID           string                 `json:"device_id"`
	DeviceType         string                 `json:"device_type"`
	DeviceName         string                 `json:"device_name"`
	DeviceOS           string                 `json:"device_os"`
	DeviceOSVersion    string                 `json:"device_os_version"`
	DeviceModel        string                 `json:"device_model"`
	DeviceManufacturer string                 `json:"device_manufacturer"`
	IPISP              string                 `json:"ip_isp"`
	BrowserName        string                 `json:"browser_name,omitempty"`
	BrowserVersion     string                 `json:"browser_version,omitempty"`
	IPAddress          string                 `json:"ip_address"`
	City               string                 `json:"city"`
	Region             string                 `json:"region"`
	Country            string                 `json:"country"`
	Timezone           string                 `json:"timezone"`
	IsTrustedDevice    bool                   `json:"is_trusted_device"`
	IsMobile           bool                   `json:"is_mobile"`
	SessionType        models.SessionType     `json:"session_type"`
	PushEnabled        bool                   `json:"push_enabled"`
	UserAgent          string                 `json:"user_agent"`
	RevokedAt          *time.Time             `json:"revoked_at,omitempty"`
	RevokedReason      *string                `json:"revoked_reason,omitempty"`
	LastActivityAt     time.Time              `json:"last_activity_at"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CreatedAt          time.Time              `json:"created_at"`
	Metadata           map[string]interface{} `json:"metadata"`
}

func NewSessionData(session *Session) SessionData {
	return SessionData{
		ID:                 session.ID,
		UserID:             session.UserID,
		DeviceID:           session.DeviceID,
		DeviceType:         session.DeviceType,
		DeviceName:         session.DeviceName,
		DeviceOS:           session.DeviceOS,
		DeviceOSVersion:    session.DeviceOSVersion,
		DeviceModel:        session.DeviceModel,
		DeviceManufacturer: session.DeviceManufacturer,
		IPISP:              session.IPISP,
		BrowserName:        session.BrowserName,
		BrowserVersion:     session.BrowserVersion,
		IPAddress:          session.IPAddress,
		City:               session.City,
		Region:             session.Region,
		Country:            session.Country,
		Timezone:           session.Timezone,
		IsTrustedDevice:    session.IsTrustedDevice,
		IsMobile:           session.IsMobile,
		SessionType:        session.SessionType,
		PushEnabled:        session.PushEnabled,
		UserAgent:          session.UserAgent,
		RevokedAt:          session.RevokedAt,
		RevokedReason:      session.RevokedReason,
		LastActivityAt:     session.LastActivityAt,
		ExpiresAt:          session.ExpiresAt,
		CreatedAt:          session.CreatedAt,
		Metadata:           session.Metadata,
	}
}

func NewSessionsData(sessions []Session) []SessionData {
	data := make([]SessionData, len(sessions))
	for i, session := range sessions {
		data[i] = NewSessionData(&session)
	}
	return data
}
