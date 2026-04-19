package domain

import (
	"time"
)

// Profile represents the profile domain model with database fields
type Profile struct {
	UserID             string
	Username           string
	DisplayName        *string
	FirstName          *string
	LastName           *string
	Bio                *string
	BioLinks           *[]string
	AvatarURL          *string
	AvatarThumbnailURL *string
	LanguageCode       string
	Timezone           *string
	CountryCode        *string
	PhoneVisible       bool
	EmailVisible       bool
	OnlineStatus       *OnlineStatus
	LastSeenAt         *time.Time
	ProfileVisibility  ProfileVisibility
	SearchVisibility   bool
	IsVerified         bool

	// enriched fields from auth.users
	PhoneVerified    bool
	EmailVerified    bool
	TwoFactorEnabled bool
	AccountStatus    AccountStatus

	// enriched fields from followers service
	IsContact *bool
}

// CreateProfileInput represents the input for creating a profile
type CreateProfileInput struct {
	DisplayName       string
	FirstName         string
	LastName          string
	Bio               string
	LanguageCode      string
	Timezone          string
	CountryCode       string
	City              string
	PhoneVisible      bool
	EmailVisible      bool
	ProfileVisibility ProfileVisibility
	SearchVisibility  bool
}

// UpdateProfileInput represents the input for updating a profile
type UpdateProfileInput struct {
	DisplayName       *string
	FirstName         *string
	LastName          *string
	Bio               *string
	BioLinks          *[]string
	DateOfBirth       *time.Time
	Gender            *string
	Pronouns          *string
	LanguageCode      *string
	Timezone          *string
	CountryCode       *string
	City              *string
	WebsiteURL        *string
	SocialLinks       *[]string
	Interests         *[]string
	PhoneVisible      *bool
	EmailVisible      *bool
	ProfileVisibility *ProfileVisibility
	SearchVisibility  *bool
}
