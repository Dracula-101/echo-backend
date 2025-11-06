package models

import (
	"database/sql/driver"
	"fmt"
)

// AccountStatus represents the status of a user account
type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusPending     AccountStatus = "pending"
	AccountStatusSuspended   AccountStatus = "suspended"
	AccountStatusLocked      AccountStatus = "locked"
	AccountStatusDeactivated AccountStatus = "deactivated"
	AccountStatusDeleted     AccountStatus = "deleted"
)

func (s AccountStatus) IsValid() bool {
	switch s {
	case AccountStatusActive, AccountStatusPending, AccountStatusSuspended,
		AccountStatusLocked, AccountStatusDeactivated, AccountStatusDeleted:
		return true
	}
	return false
}

func (s AccountStatus) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid account status: %s", s)
	}
	return string(s), nil
}

func (s *AccountStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan AccountStatus: expected string, got %T", value)
	}
	*s = AccountStatus(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid account status value: %s", str)
	}
	return nil
}

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeWeb     SessionType = "web"
	SessionTypeMobile  SessionType = "mobile"
	SessionTypeDesktop SessionType = "desktop"
	SessionTypeAPI     SessionType = "api"
	SessionTypeService SessionType = "service"
)

func (s SessionType) IsValid() bool {
	switch s {
	case SessionTypeWeb, SessionTypeMobile, SessionTypeDesktop, SessionTypeAPI, SessionTypeService:
		return true
	}
	return false
}

func (s SessionType) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid session type: %s", s)
	}
	return string(s), nil
}

func (s *SessionType) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan SessionType: expected string, got %T", value)
	}
	*s = SessionType(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid session type value: %s", str)
	}
	return nil
}

// IdentifierType represents the type of identifier used for OTP verification
type IdentifierType string

const (
	IdentifierTypeEmail    IdentifierType = "email"
	IdentifierTypePhone    IdentifierType = "phone"
	IdentifierTypeUsername IdentifierType = "username"
)

func (i IdentifierType) IsValid() bool {
	switch i {
	case IdentifierTypeEmail, IdentifierTypePhone, IdentifierTypeUsername:
		return true
	}
	return false
}

func (i IdentifierType) Value() (driver.Value, error) {
	if !i.IsValid() {
		return nil, fmt.Errorf("invalid identifier type: %s", i)
	}
	return string(i), nil
}

func (i *IdentifierType) Scan(value interface{}) error {
	if value == nil {
		*i = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan IdentifierType: expected string, got %T", value)
	}
	*i = IdentifierType(str)
	if !i.IsValid() {
		return fmt.Errorf("invalid identifier type value: %s", str)
	}
	return nil
}

// OTPPurpose represents the purpose of OTP verification
type OTPPurpose string

const (
	OTPPurposeEmailVerification OTPPurpose = "email_verification"
	OTPPurposePhoneVerification OTPPurpose = "phone_verification"
	OTPPurposePasswordReset     OTPPurpose = "password_reset"
	OTPPurposeLogin             OTPPurpose = "login"
	OTPPurposeTwoFactor         OTPPurpose = "two_factor"
	OTPPurposeAccountRecovery   OTPPurpose = "account_recovery"
)

func (o OTPPurpose) IsValid() bool {
	switch o {
	case OTPPurposeEmailVerification, OTPPurposePhoneVerification, OTPPurposePasswordReset,
		OTPPurposeLogin, OTPPurposeTwoFactor, OTPPurposeAccountRecovery:
		return true
	}
	return false
}

func (o OTPPurpose) Value() (driver.Value, error) {
	if !o.IsValid() {
		return nil, fmt.Errorf("invalid OTP purpose: %s", o)
	}
	return string(o), nil
}

func (o *OTPPurpose) Scan(value interface{}) error {
	if value == nil {
		*o = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan OTPPurpose: expected string, got %T", value)
	}
	*o = OTPPurpose(str)
	if !o.IsValid() {
		return fmt.Errorf("invalid OTP purpose value: %s", str)
	}
	return nil
}

// OAuthProvider represents supported OAuth providers
type OAuthProviderType string

const (
	OAuthProviderGoogle    OAuthProviderType = "google"
	OAuthProviderFacebook  OAuthProviderType = "facebook"
	OAuthProviderApple     OAuthProviderType = "apple"
	OAuthProviderGithub    OAuthProviderType = "github"
	OAuthProviderTwitter   OAuthProviderType = "twitter"
	OAuthProviderMicrosoft OAuthProviderType = "microsoft"
)

func (o OAuthProviderType) IsValid() bool {
	switch o {
	case OAuthProviderGoogle, OAuthProviderFacebook, OAuthProviderApple,
		OAuthProviderGithub, OAuthProviderTwitter, OAuthProviderMicrosoft:
		return true
	}
	return false
}

func (o OAuthProviderType) Value() (driver.Value, error) {
	if !o.IsValid() {
		return nil, fmt.Errorf("invalid OAuth provider: %s", o)
	}
	return string(o), nil
}

func (o *OAuthProviderType) Scan(value interface{}) error {
	if value == nil {
		*o = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan OAuthProviderType: expected string, got %T", value)
	}
	*o = OAuthProviderType(str)
	if !o.IsValid() {
		return fmt.Errorf("invalid OAuth provider value: %s", str)
	}
	return nil
}

// SecurityEventType represents the type of security event
type SecurityEventType string

const (
	SecurityEventLogin              SecurityEventType = "login"
	SecurityEventLogout             SecurityEventType = "logout"
	SecurityEventLoginFailed        SecurityEventType = "login_failed"
	SecurityEventPasswordChange     SecurityEventType = "password_change"
	SecurityEventPasswordReset      SecurityEventType = "password_reset"
	SecurityEventTwoFactorEnabled   SecurityEventType = "two_factor_enabled"
	SecurityEventTwoFactorDisabled  SecurityEventType = "two_factor_disabled"
	SecurityEventAccountLocked      SecurityEventType = "account_locked"
	SecurityEventAccountUnlocked    SecurityEventType = "account_unlocked"
	SecurityEventSuspiciousActivity SecurityEventType = "suspicious_activity"
	SecurityEventUnauthorizedAccess SecurityEventType = "unauthorized_access"
	SecurityEventSessionRevoked     SecurityEventType = "session_revoked"
)

func (s SecurityEventType) IsValid() bool {
	switch s {
	case SecurityEventLogin, SecurityEventLogout, SecurityEventLoginFailed,
		SecurityEventPasswordChange, SecurityEventPasswordReset,
		SecurityEventTwoFactorEnabled, SecurityEventTwoFactorDisabled,
		SecurityEventAccountLocked, SecurityEventAccountUnlocked,
		SecurityEventSuspiciousActivity, SecurityEventUnauthorizedAccess,
		SecurityEventSessionRevoked:
		return true
	}
	return false
}

func (s SecurityEventType) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid security event type: %s", s)
	}
	return string(s), nil
}

func (s *SecurityEventType) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan SecurityEventType: expected string, got %T", value)
	}
	*s = SecurityEventType(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid security event type value: %s", str)
	}
	return nil
}

// SecuritySeverity represents the severity level of a security event
type SecuritySeverity string

const (
	SecuritySeverityLow      SecuritySeverity = "low"
	SecuritySeverityMedium   SecuritySeverity = "medium"
	SecuritySeverityHigh     SecuritySeverity = "high"
	SecuritySeverityCritical SecuritySeverity = "critical"
)

func (s SecuritySeverity) IsValid() bool {
	switch s {
	case SecuritySeverityLow, SecuritySeverityMedium, SecuritySeverityHigh, SecuritySeverityCritical:
		return true
	}
	return false
}

func (s SecuritySeverity) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid security severity: %s", s)
	}
	return string(s), nil
}

func (s *SecuritySeverity) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan SecuritySeverity: expected string, got %T", value)
	}
	*s = SecuritySeverity(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid security severity value: %s", str)
	}
	return nil
}

// RateLimitActionType represents the type of action being rate limited
type RateLimitActionType string

const (
	RateLimitActionLogin         RateLimitActionType = "login"
	RateLimitActionOTP           RateLimitActionType = "otp"
	RateLimitActionPasswordReset RateLimitActionType = "password_reset"
	RateLimitActionAPI           RateLimitActionType = "api"
	RateLimitActionMessage       RateLimitActionType = "message"
)

func (r RateLimitActionType) IsValid() bool {
	switch r {
	case RateLimitActionLogin, RateLimitActionOTP, RateLimitActionPasswordReset,
		RateLimitActionAPI, RateLimitActionMessage:
		return true
	}
	return false
}

func (r RateLimitActionType) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid rate limit action type: %s", r)
	}
	return string(r), nil
}

func (r *RateLimitActionType) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan RateLimitActionType: expected string, got %T", value)
	}
	*r = RateLimitActionType(str)
	if !r.IsValid() {
		return fmt.Errorf("invalid rate limit action type value: %s", str)
	}
	return nil
}
