package domain

import (
	"time"

	"shared/pkg/database/postgres/models"
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
		return ErrAccountLocked
	}

	switch u.AccountStatus {
	case models.AccountStatusDeactivated:
		return ErrAccountDeactivated
	case models.AccountStatusSuspended:
		return ErrAccountSuspended
	case models.AccountStatusPending:
		return ErrAccountPending
	case models.AccountStatusDeleted:
		return ErrAccountDeleted
	case models.AccountStatusActive:
		return nil
	default:
		return ErrUnknownAccountStatus
	}
}
