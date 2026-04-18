package dto

// GetProfileRequest represents a request to get a user profile
type GetProfileRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// GetProfileResponse represents the response for getting a user profile
type GetProfileResponse struct {
	ID                 string  `json:"id"`
	Username           string  `json:"username"`
	DisplayName        *string `json:"display_name"`
	FirstName          *string `json:"first_name"`
	LastName           *string `json:"last_name"`
	Bio                *string `json:"bio"`
	AvatarURL          *string `json:"avatar_url"`
	AvatarThumbnailURL *string `json:"avatar_thumbnail_url"`
	LanguageCode       string  `json:"language_code"`
	Timezone           *string `json:"timezone"`
	CountryCode        *string `json:"country_code"`
	IsVerified         bool    `json:"is_verified"`
	PhoneVerified      bool    `json:"phone_verified"`
	EmailVerified      bool    `json:"email_verified"`
	TwoFactorEnabled   bool    `json:"two_factor_enabled"`
	AccountStatus      string  `json:"account_status"`
}
