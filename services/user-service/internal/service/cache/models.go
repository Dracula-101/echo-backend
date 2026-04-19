package cache

import "user-service/internal/domain"

type Settings struct {
	UserID                  string `json:"user_id"`
	ProfileVisibility       string `json:"profile_visibility"`
	LastSeenVisibility      string `json:"last_seen_visibility"`
	OnlineStatusVisibility  string `json:"online_status_visibility"`
	ProfilePhotoVisibility  string `json:"profile_photo_visibility"`
	AboutVisibility         string `json:"about_visibility"`
	ReadReceiptsEnabled     bool   `json:"read_receipts_enabled"`
	TypingIndicatorsEnabled bool   `json:"typing_indicators_enabled"`
	UpdatedAt               int64  `json:"updated_at"`
}

type Presence struct {
	Status       string `json:"status"`
	LastSeenUnix int64  `json:"last_seen_unix"`
	DeviceCount  int    `json:"device_count"`
}

type ContactPresenceEntry struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

type StatusEntry struct {
	StatusID  string `json:"status_id"`
	UserID    string `json:"user_id"`
	MediaURL  string `json:"media_url"`
	Caption   string `json:"caption"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type SearchResult struct {
	Users  []*domain.Profile `json:"users"`
	Total  int64             `json:"total"`
	Offset int64             `json:"offset"`
}

type PrivacyResult struct {
	CanSeeProfile      bool `json:"can_see_profile"`
	CanSeeAvatar       bool `json:"can_see_avatar"`
	CanSeeLastSeen     bool `json:"can_see_last_seen"`
	CanSeeOnlineStatus bool `json:"can_see_online_status"`
	CanSeeAbout        bool `json:"can_see_about"`
	CanSeePhone        bool `json:"can_see_phone"`
	CanSeeEmail        bool `json:"can_see_email"`
	AreContacts        bool `json:"are_contacts"`
}
