package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type AuthUser struct {
	ID                     string           `db:"id" json:"id" pk:"true"`
	Email                  string           `db:"email" json:"email"`
	PhoneNumber            *string          `db:"phone_number" json:"phone_number,omitempty"`
	PhoneCountryCode       *string          `db:"phone_country_code" json:"phone_country_code,omitempty"`
	EmailVerified          bool             `db:"email_verified" json:"email_verified"`
	PhoneVerified          bool             `db:"phone_verified" json:"phone_verified"`
	PasswordHash           string           `db:"password_hash" json:"-"`
	PasswordSalt           string           `db:"password_salt" json:"-"`
	PasswordAlgorithm      string           `db:"password_algorithm" json:"password_algorithm"`
	PasswordLastChangedAt  *time.Time       `db:"password_last_changed_at" json:"password_last_changed_at,omitempty"`
	TwoFactorEnabled       bool             `db:"two_factor_enabled" json:"two_factor_enabled"`
	TwoFactorSecret        *string          `db:"two_factor_secret" json:"-"`
	TwoFactorBackupCodes   pq.StringArray   `db:"two_factor_backup_codes" json:"two_factor_backup_codes,omitempty"`
	AccountStatus          AccountStatus    `db:"account_status" json:"account_status"`
	AccountLockedUntil     *time.Time       `db:"account_locked_until" json:"account_locked_until,omitempty"`
	FailedLoginAttempts    int              `db:"failed_login_attempts" json:"failed_login_attempts"`
	LastFailedLoginAt      *time.Time       `db:"last_failed_login_at" json:"last_failed_login_at,omitempty"`
	LastSuccessfulLoginAt  *time.Time       `db:"last_successful_login_at" json:"last_successful_login_at,omitempty"`
	RequiresPasswordChange bool             `db:"requires_password_change" json:"requires_password_change"`
	PasswordHistory        json.RawMessage  `db:"password_history" json:"password_history,omitempty"`
	SecurityQuestions      *json.RawMessage `db:"security_questions" json:"security_questions,omitempty"`
	CreatedAt              time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time        `db:"updated_at" json:"updated_at"`
	DeletedAt              *time.Time       `db:"deleted_at" json:"deleted_at,omitempty"`
	CreatedByIP            *string          `db:"created_by_ip" json:"created_by_ip,omitempty"`
	CreatedByUserAgent     *string          `db:"created_by_user_agent" json:"created_by_user_agent,omitempty"`
}

func (a *AuthUser) TableName() string {
	return "auth.users"
}

func (a *AuthUser) PrimaryKey() interface{} {
	return a.ID
}

type AuthSession struct {
	ID                 string           `db:"id" json:"id" pk:"true"`
	UserID             string           `db:"user_id" json:"user_id"`
	SessionToken       string           `db:"session_token" json:"session_token"`
	RefreshToken       *string          `db:"refresh_token" json:"refresh_token,omitempty"`
	DeviceID           *string          `db:"device_id" json:"device_id,omitempty"`
	DeviceName         *string          `db:"device_name" json:"device_name,omitempty"`
	DeviceType         *string          `db:"device_type" json:"device_type,omitempty"`
	DeviceOS           *string          `db:"device_os" json:"device_os,omitempty"`
	DeviceOSVersion    *string          `db:"device_os_version" json:"device_os_version,omitempty"`
	DeviceModel        *string          `db:"device_model" json:"device_model,omitempty"`
	DeviceManufacturer *string          `db:"device_manufacturer" json:"device_manufacturer,omitempty"`
	BrowserName        *string          `db:"browser_name" json:"browser_name,omitempty"`
	BrowserVersion     *string          `db:"browser_version" json:"browser_version,omitempty"`
	UserAgent          *string          `db:"user_agent" json:"user_agent,omitempty"`
	IPAddress          string           `db:"ip_address" json:"ip_address"`
	IPCountry          *string          `db:"ip_country" json:"ip_country,omitempty"`
	IPRegion           *string          `db:"ip_region" json:"ip_region,omitempty"`
	IPCity             *string          `db:"ip_city" json:"ip_city,omitempty"`
	IPTimezone         *string          `db:"ip_timezone" json:"ip_timezone,omitempty"`
	IPISP              *string          `db:"ip_isp" json:"ip_isp,omitempty"`
	Latitude           *float64         `db:"latitude" json:"latitude,omitempty"`
	Longitude          *float64         `db:"longitude" json:"longitude,omitempty"`
	IsMobile           bool             `db:"is_mobile" json:"is_mobile"`
	IsTrustedDevice    bool             `db:"is_trusted_device" json:"is_trusted_device"`
	FCMToken           *string          `db:"fcm_token" json:"fcm_token,omitempty"`
	APNSToken          *string          `db:"apns_token" json:"apns_token,omitempty"`
	PushEnabled        bool             `db:"push_enabled" json:"push_enabled"`
	SessionType        SessionType      `db:"session_type" json:"session_type"`
	ExpiresAt          time.Time        `db:"expires_at" json:"expires_at"`
	LastActivityAt     time.Time        `db:"last_activity_at" json:"last_activity_at"`
	LastRefreshAt      *time.Time       `db:"last_refresh_at" json:"last_refresh_at,omitempty"`
	CreatedAt          time.Time        `db:"created_at" json:"created_at"`
	RevokedAt          *time.Time       `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedReason      *string          `db:"revoked_reason" json:"revoked_reason,omitempty"`
	Metadata           *json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (a *AuthSession) TableName() string {
	return "auth.sessions"
}

func (a *AuthSession) PrimaryKey() interface{} {
	return a.ID
}

type OTPVerification struct {
	ID             string          `db:"id" json:"id" pk:"true"`
	UserID         *string         `db:"user_id" json:"user_id,omitempty"`
	Identifier     string          `db:"identifier" json:"identifier"`
	IdentifierType IdentifierType  `db:"identifier_type" json:"identifier_type"`
	OTPCode        string          `db:"otp_code" json:"-"`
	OTPHash        string          `db:"otp_hash" json:"-"`
	Purpose        OTPPurpose      `db:"purpose" json:"purpose"`
	Attempts       int             `db:"attempts" json:"attempts"`
	MaxAttempts    int             `db:"max_attempts" json:"max_attempts"`
	IsVerified     bool            `db:"is_verified" json:"is_verified"`
	VerifiedAt     *time.Time      `db:"verified_at" json:"verified_at,omitempty"`
	ExpiresAt      time.Time       `db:"expires_at" json:"expires_at"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	SentVia        *string         `db:"sent_via" json:"sent_via,omitempty"`
	IPAddress      *string         `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent      *string         `db:"user_agent" json:"user_agent,omitempty"`
	Metadata       json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (o *OTPVerification) TableName() string {
	return "auth.otp_verifications"
}

func (o *OTPVerification) PrimaryKey() interface{} {
	return o.ID
}

type OAuthProvider struct {
	ID               string            `db:"id" json:"id" pk:"true"`
	UserID           string            `db:"user_id" json:"user_id"`
	Provider         OAuthProviderType `db:"provider" json:"provider"`
	ProviderUserID   string            `db:"provider_user_id" json:"provider_user_id"`
	ProviderEmail    *string           `db:"provider_email" json:"provider_email,omitempty"`
	ProviderUsername *string           `db:"provider_username" json:"provider_username,omitempty"`
	AccessToken      *string           `db:"access_token" json:"-"`
	RefreshToken     *string           `db:"refresh_token" json:"-"`
	TokenExpiresAt   *time.Time        `db:"token_expires_at" json:"token_expires_at,omitempty"`
	Scope            pq.StringArray    `db:"scope" json:"scope,omitempty"`
	ProfileData      json.RawMessage   `db:"profile_data" json:"profile_data,omitempty"`
	IsPrimary        bool              `db:"is_primary" json:"is_primary"`
	LinkedAt         time.Time         `db:"linked_at" json:"linked_at"`
	LastUsedAt       *time.Time        `db:"last_used_at" json:"last_used_at,omitempty"`
	UnlinkedAt       *time.Time        `db:"unlinked_at" json:"unlinked_at,omitempty"`
	CreatedAt        time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time         `db:"updated_at" json:"updated_at"`
}

func (o *OAuthProvider) TableName() string {
	return "auth.oauth_providers"
}

func (o *OAuthProvider) PrimaryKey() interface{} {
	return o.ID
}

type PasswordResetToken struct {
	ID            string     `db:"id" json:"id" pk:"true"`
	UserID        string     `db:"user_id" json:"user_id"`
	Token         string     `db:"token" json:"-"`
	TokenHash     string     `db:"token_hash" json:"-"`
	ExpiresAt     time.Time  `db:"expires_at" json:"expires_at"`
	UsedAt        *time.Time `db:"used_at" json:"used_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	IPAddress     *string    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent     *string    `db:"user_agent" json:"user_agent,omitempty"`
	EmailSentAt   *time.Time `db:"email_sent_at" json:"email_sent_at,omitempty"`
	EmailOpenedAt *time.Time `db:"email_opened_at" json:"email_opened_at,omitempty"`
	LinkClickedAt *time.Time `db:"link_clicked_at" json:"link_clicked_at,omitempty"`
}

func (p *PasswordResetToken) TableName() string {
	return "auth.password_reset_tokens"
}

func (p *PasswordResetToken) PrimaryKey() interface{} {
	return p.ID
}

type EmailVerificationToken struct {
	ID         string     `db:"id" json:"id" pk:"true"`
	UserID     string     `db:"user_id" json:"user_id"`
	Email      string     `db:"email" json:"email"`
	Token      string     `db:"token" json:"-"`
	TokenHash  string     `db:"token_hash" json:"-"`
	ExpiresAt  time.Time  `db:"expires_at" json:"expires_at"`
	VerifiedAt *time.Time `db:"verified_at" json:"verified_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	IPAddress  *string    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent  *string    `db:"user_agent" json:"user_agent,omitempty"`
	Attempts   int        `db:"attempts" json:"attempts"`
}

func (e *EmailVerificationToken) TableName() string {
	return "auth.email_verification_tokens"
}

func (e *EmailVerificationToken) PrimaryKey() interface{} {
	return e.ID
}

type SecurityEvent struct {
	ID              string            `db:"id" json:"id" pk:"true"`
	UserID          *string           `db:"user_id" json:"user_id,omitempty"`
	SessionID       *string           `db:"session_id" json:"session_id,omitempty"`
	EventType       SecurityEventType `db:"event_type" json:"event_type"`
	EventCategory   *string           `db:"event_category" json:"event_category,omitempty"`
	Severity        SecuritySeverity  `db:"severity" json:"severity"`
	Status          *string           `db:"status" json:"status,omitempty"`
	Description     *string           `db:"description" json:"description,omitempty"`
	IPAddress       *string           `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent       *string           `db:"user_agent" json:"user_agent,omitempty"`
	DeviceID        *string           `db:"device_id" json:"device_id,omitempty"`
	LocationCountry *string           `db:"location_country" json:"location_country,omitempty"`
	LocationCity    *string           `db:"location_city" json:"location_city,omitempty"`
	RiskScore       *int              `db:"risk_score" json:"risk_score,omitempty"`
	IsSuspicious    bool              `db:"is_suspicious" json:"is_suspicious"`
	BlockedReason   *string           `db:"blocked_reason" json:"blocked_reason,omitempty"`
	CreatedAt       time.Time         `db:"created_at" json:"created_at"`
	Metadata        *json.RawMessage  `db:"metadata" json:"metadata,omitempty"`
}

func (s *SecurityEvent) TableName() string {
	return "auth.security_events"
}

func (s *SecurityEvent) PrimaryKey() interface{} {
	return s.ID
}

type LoginHistory struct {
	ID                string    `db:"id" json:"id" pk:"true"`
	UserID            string    `db:"user_id" json:"user_id"`
	SessionID         *string   `db:"session_id" json:"session_id,omitempty"`
	LoginMethod       *string   `db:"login_method" json:"login_method,omitempty"`
	Status            *string   `db:"status" json:"status,omitempty"`
	FailureReason     *string   `db:"failure_reason" json:"failure_reason,omitempty"`
	IPAddress         *string   `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent         *string   `db:"user_agent" json:"user_agent,omitempty"`
	DeviceID          *string   `db:"device_id" json:"device_id,omitempty"`
	DeviceFingerprint *string   `db:"device_fingerprint" json:"device_fingerprint,omitempty"`
	LocationCountry   *string   `db:"location_country" json:"location_country,omitempty"`
	LocationCity      *string   `db:"location_city" json:"location_city,omitempty"`
	Latitude          *float64  `db:"latitude" json:"latitude,omitempty"`
	Longitude         *float64  `db:"longitude" json:"longitude,omitempty"`
	IsNewDevice       bool      `db:"is_new_device" json:"is_new_device"`
	IsNewLocation     bool      `db:"is_new_location" json:"is_new_location"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

func (l *LoginHistory) TableName() string {
	return "auth.login_history"
}

func (l *LoginHistory) PrimaryKey() interface{} {
	return l.ID
}

type APIKey struct {
	ID               string          `db:"id" json:"id" pk:"true"`
	KeyName          string          `db:"key_name" json:"key_name"`
	KeyHash          string          `db:"key_hash" json:"-"`
	KeyPrefix        string          `db:"key_prefix" json:"key_prefix"`
	UserID           *string         `db:"user_id" json:"user_id,omitempty"`
	ServiceName      *string         `db:"service_name" json:"service_name,omitempty"`
	Scopes           pq.StringArray  `db:"scopes" json:"scopes,omitempty"`
	RateLimitPerHour int             `db:"rate_limit_per_hour" json:"rate_limit_per_hour"`
	IsActive         bool            `db:"is_active" json:"is_active"`
	ExpiresAt        *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	LastUsedAt       *time.Time      `db:"last_used_at" json:"last_used_at,omitempty"`
	Description      *string         `db:"description" json:"description,omitempty"`
	Metadata         json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
}

func (a *APIKey) TableName() string {
	return "auth.api_keys"
}

func (a *APIKey) PrimaryKey() interface{} {
	return a.ID
}
