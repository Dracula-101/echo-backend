package dto

import (
	"encoding/json"
	"time"
)

// GetProfileRequest represents a request to get a user profile
type GetProfileRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// GetProfileResponse represents the response for getting a user profile
type GetProfileResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  *string   `json:"display_name"`
	FirstName    *string   `json:"first_name"`
	LastName     *string   `json:"last_name"`
	Bio          *string   `json:"bio"`
	AvatarURL    *string   `json:"avatar_url"`
	LanguageCode string    `json:"language_code"`
	Timezone     *string   `json:"timezone"`
	CountryCode  *string   `json:"country_code"`
	IsVerified   bool      `json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MarshalJSON implements custom JSON marshaling to preserve field order
func (r *GetProfileResponse) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 512)
	buf = append(buf, '{')

	// ID
	buf = append(buf, `"id":`...)
	idJSON, _ := json.Marshal(r.ID)
	buf = append(buf, idJSON...)

	// Username
	buf = append(buf, `,"username":`...)
	usernameJSON, _ := json.Marshal(r.Username)
	buf = append(buf, usernameJSON...)

	// DisplayName
	if r.DisplayName != nil {
		buf = append(buf, `,"display_name":`...)
		displayNameJSON, _ := json.Marshal(r.DisplayName)
		buf = append(buf, displayNameJSON...)
	}

	// FirstName
	if r.FirstName != nil {
		buf = append(buf, `,"first_name":`...)
		firstNameJSON, _ := json.Marshal(r.FirstName)
		buf = append(buf, firstNameJSON...)
	}

	// LastName
	if r.LastName != nil {
		buf = append(buf, `,"last_name":`...)
		lastNameJSON, _ := json.Marshal(r.LastName)
		buf = append(buf, lastNameJSON...)
	}

	// Bio
	if r.Bio != nil {
		buf = append(buf, `,"bio":`...)
		bioJSON, _ := json.Marshal(r.Bio)
		buf = append(buf, bioJSON...)
	}

	// AvatarURL
	if r.AvatarURL != nil {
		buf = append(buf, `,"avatar_url":`...)
		avatarJSON, _ := json.Marshal(r.AvatarURL)
		buf = append(buf, avatarJSON...)
	}

	// LanguageCode
	buf = append(buf, `,"language_code":`...)
	langJSON, _ := json.Marshal(r.LanguageCode)
	buf = append(buf, langJSON...)

	// Timezone
	if r.Timezone != nil {
		buf = append(buf, `,"timezone":`...)
		timezoneJSON, _ := json.Marshal(r.Timezone)
		buf = append(buf, timezoneJSON...)
	}

	// CountryCode
	if r.CountryCode != nil {
		buf = append(buf, `,"country_code":`...)
		countryJSON, _ := json.Marshal(r.CountryCode)
		buf = append(buf, countryJSON...)
	}

	// IsVerified
	buf = append(buf, `,"is_verified":`...)
	verifiedJSON, _ := json.Marshal(r.IsVerified)
	buf = append(buf, verifiedJSON...)

	// CreatedAt
	buf = append(buf, `,"created_at":`...)
	createdJSON, _ := json.Marshal(r.CreatedAt)
	buf = append(buf, createdJSON...)

	// UpdatedAt
	buf = append(buf, `,"updated_at":`...)
	updatedJSON, _ := json.Marshal(r.UpdatedAt)
	buf = append(buf, updatedJSON...)

	buf = append(buf, '}')
	return buf, nil
}
