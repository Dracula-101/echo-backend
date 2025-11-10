package dto

import "time"

// UpdateProfileRequest represents a request to update a user profile
type UpdateProfileRequest struct {
	Username     *string `json:"username,omitempty" validate:"omitempty,min=3,max=30,alphanum"`
	DisplayName  *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	FirstName    *string `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName     *string `json:"last_name,omitempty" validate:"omitempty,max=50"`
	Bio          *string `json:"bio,omitempty" validate:"omitempty,max=500"`
	AvatarURL    *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
	LanguageCode *string `json:"language_code,omitempty" validate:"omitempty,len=2"`
	Timezone     *string `json:"timezone,omitempty"`
	CountryCode  *string `json:"country_code,omitempty" validate:"omitempty,len=2"`
}

// UpdateProfileResponse represents the response for updating a user profile
type UpdateProfileResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  *string   `json:"display_name,omitempty"`
	FirstName    *string   `json:"first_name,omitempty"`
	LastName     *string   `json:"last_name,omitempty"`
	Bio          *string   `json:"bio,omitempty"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	LanguageCode string    `json:"language_code"`
	Timezone     *string   `json:"timezone,omitempty"`
	CountryCode  *string   `json:"country_code,omitempty"`
	IsVerified   bool      `json:"is_verified"`
	UpdatedAt    time.Time `json:"updated_at"`
}
