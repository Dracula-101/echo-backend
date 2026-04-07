package domain

import (
	"time"

	"shared/pkg/database/postgres/models"
	"shared/pkg/utils"

	authError "auth-service/internal/error"
)

// User represents the core user entity in the authentication domain
type User struct {
	ID                     string
	Email                  string
	PhoneNumber            string
	PhoneCountryCode       string
	EmailVerified          bool
	PhoneVerified          bool
	AccountStatus          models.AccountStatus
	TwoFactorEnabled       bool
	TwoFactorSecret        string
	PasswordHash           string
	PasswordSalt           string
	PasswordAlgorithm      string
	PasswordLastChanged    *time.Time
	AccountLockedUntil     *time.Time
	FailedLoginAttempts    int
	LastFailedLoginAt      *time.Time
	LastSuccessfulLoginAt  *time.Time
	RequiresPasswordChange bool
	DeletedAt              *time.Time
	UpdatedAt              time.Time
	CreatedAt              time.Time
}

// IsLocked checks if the user account is currently locked
func (u *User) IsLocked() bool {
	if u.AccountLockedUntil == nil {
		return false
	}
	return u.AccountLockedUntil.After(time.Now())
}

// IsActive checks if the user account is active and can login
func (u *User) IsActive() bool {
	return u.AccountStatus == models.AccountStatusActive && !u.IsLocked()
}

// CanLogin validates if user can proceed with login
func (u *User) CanLogin() error {
	if u.IsLocked() {
		return authError.ErrAccountLocked
	}

	switch u.AccountStatus {
	case models.AccountStatusDeactivated:
		return authError.ErrAccountDeactivated
	case models.AccountStatusSuspended:
		return authError.ErrAccountSuspended
	case models.AccountStatusPending:
		return authError.ErrAccountPending
	case models.AccountStatusDeleted:
		return authError.ErrAccountDeleted
	case models.AccountStatusActive:
		return nil
	default:
		return authError.ErrUnknownAccountStatus
	}
}

func toUserFromModel(m *models.AuthUser) *User {
	if m == nil {
		return nil
	}
	return &User{
		ID:                     m.ID,
		Email:                  m.Email,
		PhoneNumber:            utils.DerefString(m.PhoneNumber),
		PhoneCountryCode:       utils.DerefString(m.PhoneCountryCode),
		EmailVerified:          m.EmailVerified,
		PhoneVerified:          m.PhoneVerified,
		AccountStatus:          m.AccountStatus,
		TwoFactorEnabled:       m.TwoFactorEnabled,
		TwoFactorSecret:        utils.DerefString(m.TwoFactorSecret),
		PasswordHash:           m.PasswordHash,
		PasswordSalt:           m.PasswordSalt,
		PasswordAlgorithm:      m.PasswordAlgorithm,
		PasswordLastChanged:    m.PasswordLastChangedAt,
		AccountLockedUntil:     m.AccountLockedUntil,
		FailedLoginAttempts:    m.FailedLoginAttempts,
		LastFailedLoginAt:      m.LastFailedLoginAt,
		LastSuccessfulLoginAt:  m.LastSuccessfulLoginAt,
		RequiresPasswordChange: m.RequiresPasswordChange,
		DeletedAt:              m.DeletedAt,
		UpdatedAt:              m.UpdatedAt,
		CreatedAt:              m.CreatedAt,
	}
}
