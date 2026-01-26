package domain

import "time"

// Profile represents the profile domain model with database fields
type Profile struct {
	ID                string     `db:"id"`
	UserID            string     `db:"user_id"`
	Username          string     `db:"username"`
	DisplayName       *string    `db:"display_name"`
	FirstName         *string    `db:"first_name"`
	LastName          *string    `db:"last_name"`
	Bio               *string    `db:"bio"`
	AvatarURL         *string    `db:"avatar_url"`
	LanguageCode      string     `db:"language_code"`
	Timezone          *string    `db:"timezone"`
	CountryCode       *string    `db:"country_code"`
	PhoneVisible      bool       `db:"phone_visible"`
	EmailVisible      bool       `db:"email_visible"`
	OnlineStatus      string     `db:"online_status"`
	LastSeenAt        *time.Time `db:"last_seen_at"`
	ProfileVisibility string     `db:"profile_visibility"`
	SearchVisibility  bool       `db:"search_visibility"`
	IsVerified        bool       `db:"is_verified"`
	CreatedAt         *time.Time `db:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at"`
	DeactivatedAt     *time.Time `db:"deactivated_at"`
}

// CreateProfileInput represents the input for creating a profile
type CreateProfileInput struct {
	UserID       string
	Username     string
	DisplayName  string
	FirstName    *string
	LastName     *string
	Bio          *string
	AvatarURL    *string
	LanguageCode *string
	Timezone     *string
	CountryCode  *string
	IsVerified   bool
}

// UpdateProfileInput represents the input for updating a profile
type UpdateProfileInput struct {
	UserID       string
	Username     *string
	DisplayName  *string
	FirstName    *string
	LastName     *string
	Bio          *string
	AvatarURL    *string
	LanguageCode *string
	Timezone     *string
	CountryCode  *string
}
