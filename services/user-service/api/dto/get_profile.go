package dto

import "time"

// GetProfileRequest represents a request to get a user profile
type GetProfileRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// GetProfileResponse represents the response for getting a user profile
type GetProfileResponse struct {
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
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
