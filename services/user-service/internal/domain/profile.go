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
	DeactivatedAt      *time.Time

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

type SearchProfile struct {
	UserID           string     `json:"user_id"`
	Username         string     `json:"username"`
	DisplayName      *string    `json:"display_name"`
	FirstName        *string    `json:"first_name"`
	LastName         *string    `json:"last_name"`
	SearchVisibility bool       `json:"search_visibility"`
	DeactivatedAt    *time.Time `json:"deactivated_at"`
}

// SearchID implements [search.Document].
func (s SearchProfile) SearchID() string {
	return s.UserID
}

// SearchIndex implements [search.Document].
func (s SearchProfile) SearchIndex() string {
	return "users"
}
