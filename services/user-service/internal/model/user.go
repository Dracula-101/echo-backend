package model

import "time"

// User represents the user domain model
type User struct {
	ID           string
	Username     string
	DisplayName  *string
	FirstName    *string
	LastName     *string
	Bio          *string
	AvatarURL    *string
	Email        string
	PhoneNumber  *string
	LanguageCode string
	Timezone     *string
	CountryCode  *string
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ProfileUpdate represents a profile update request
type ProfileUpdate struct {
	Username    *string
	DisplayName *string
	FirstName   *string
	LastName    *string
	Bio         *string
	AvatarURL   *string
	LanguageCode *string
	Timezone    *string
	CountryCode *string
}
