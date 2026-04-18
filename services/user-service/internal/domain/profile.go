package domain

import "time"

// Profile represents the profile domain model with database fields
type Profile struct {
	UserID             string       `json:"user_id"`
	Username           string       `json:"username"`
	DisplayName        *string      `json:"display_name"`
	FirstName          *string      `json:"first_name,omitempty"`
	LastName           *string      `json:"last_name,omitempty"`
	Bio                *string      `json:"bio,omitempty"`
	AvatarURL          *string      `json:"avatar_url"`
	AvatarThumbnailURL *string      `json:"avatar_thumbnail_url"`
	LanguageCode       string       `json:"language_code"`
	Timezone           *string      `json:"timezone"`
	CountryCode        *string      `json:"country_code"`
	PhoneVisible       bool         `json:"phone_visible"`
	EmailVisible       bool         `json:"email_visible"`
	OnlineStatus       OnlineStatus `json:"online_status"`
	LastSeenAt         *time.Time   `json:"last_seen_at"`
	ProfileVisibility  string       `json:"profile_visibility"`
	SearchVisibility   bool         `json:"search_visibility"`
	IsVerified         bool         `json:"is_verified"`

	// enriched fields from auth.users
	PhoneVerified    bool          `json:"phone_verified"`
	EmailVerified    bool          `json:"email_verified"`
	TwoFactorEnabled bool          `json:"two_factor_enabled"`
	AccountStatus    AccountStatus `json:"account_status"`
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
	Searchable   bool
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

// ProfileUpdate represents a profile update request
type ProfileUpdate struct {
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
