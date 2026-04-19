package dto

import (
	"time"
	"user-service/internal/domain"
)

// GetProfileResponse represents the response for getting a user profile
type GetProfileResponse struct {
	UserID             string     `json:"user_id"`
	Username           string     `json:"username"`
	DisplayName        *string    `json:"display_name,omitempty"`
	FirstName          *string    `json:"first_name,omitempty"`
	LastName           *string    `json:"last_name,omitempty"`
	Bio                *string    `json:"bio,omitempty"`
	AvatarURL          *string    `json:"avatar_url,omitempty"`
	AvatarThumbnailURL *string    `json:"avatar_thumbnail_url,omitempty"`
	LanguageCode       string     `json:"language_code"`
	Timezone           *string    `json:"timezone,omitempty"`
	CountryCode        *string    `json:"country_code,omitempty"`
	PhoneVisible       bool       `json:"phone_visible"`
	EmailVisible       bool       `json:"email_visible"`
	OnlineStatus       string     `json:"online_status"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
	ProfileVisibility  string     `json:"profile_visibility"`
	SearchVisibility   bool       `json:"search_visibility"`
	IsVerified         bool       `json:"is_verified"`

	// enriched fields from auth.users
	PhoneVerified    bool   `json:"phone_verified"`
	EmailVerified    bool   `json:"email_verified"`
	TwoFactorEnabled bool   `json:"two_factor_enabled"`
	AccountStatus    string `json:"account_status"`
}

func NewGetProfileResponse(profile *domain.Profile) *GetProfileResponse {
	return &GetProfileResponse{
		UserID:             profile.UserID,
		Username:           profile.Username,
		DisplayName:        profile.DisplayName,
		FirstName:          profile.FirstName,
		LastName:           profile.LastName,
		Bio:                profile.Bio,
		AvatarURL:          profile.AvatarURL,
		AvatarThumbnailURL: profile.AvatarThumbnailURL,
		LanguageCode:       profile.LanguageCode,
		Timezone:           profile.Timezone,
		CountryCode:        profile.CountryCode,
		PhoneVisible:       profile.PhoneVisible,
		EmailVisible:       profile.EmailVisible,
		OnlineStatus:       profile.OnlineStatus.String(),
		LastSeenAt:         profile.LastSeenAt,
		ProfileVisibility:  profile.ProfileVisibility.String(),
		SearchVisibility:   profile.SearchVisibility,
		IsVerified:         profile.IsVerified,
		PhoneVerified:      profile.PhoneVerified,
		EmailVerified:      profile.EmailVerified,
		TwoFactorEnabled:   profile.TwoFactorEnabled,
		AccountStatus:      profile.AccountStatus.String(),
	}
}
