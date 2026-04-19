package domain

import "time"

// Profile represents the profile domain model with database fields
type Profile struct {
	UserID             string
	Username           string
	DisplayName        *string
	FirstName          *string
	LastName           *string
	Bio                *string
	AvatarURL          *string
	AvatarThumbnailURL *string
	LanguageCode       string
	Timezone           *string
	CountryCode        *string
	PhoneVisible       bool
	EmailVisible       bool
	OnlineStatus       OnlineStatus
	LastSeenAt         *time.Time
	ProfileVisibility  ProfileVisibility
	SearchVisibility   bool
	IsVerified         bool

	// enriched fields from auth.users
	PhoneVerified    bool
	EmailVerified    bool
	TwoFactorEnabled bool
	AccountStatus    AccountStatus
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
	DisplayName       *string
	FirstName         *string
	LastName          *string
	Bio               *string
	BioLinks          *[]string
	DateOfBirth       *time.Time
	AvatarURL         *string
	LanguageCode      *string
	Timezone          *string
	CountryCode       *string
	SocialLinks       *[]string
	ProfileVisibility *ProfileVisibility
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
