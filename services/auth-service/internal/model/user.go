package model

import (
	"time"
)

type User struct {
	ID                     string     `db:"id"`
	Email                  string     `db:"email"`
	PhoneNumber            *string    `db:"phone_number"`
	PhoneCountryCode       *string    `db:"phone_country_code"`
	EmailVerified          bool       `db:"email_verified"`
	PhoneVerified          bool       `db:"phone_verified"`
	PasswordHash           string     `db:"password_hash"`
	PasswordLastChanged    *time.Time `db:"password_last_changed_at"`
	TwoFactorEnabled       bool       `db:"two_factor_enabled"`
	AccountStatus          string     `db:"account_status"`
	AccountLockedUntil     *time.Time `db:"account_locked_until"`
	FailedLoginAttempts    int        `db:"failed_login_attempts"`
	RequiresPasswordChange bool       `db:"requires_password_change"`
	CreatedAt              time.Time  `db:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at"`
	DeletedAt              *time.Time `db:"deleted_at"`
}
