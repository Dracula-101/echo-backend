package cache

import "user-service/internal/domain"

type Settings struct {
	UserID               string `json:"user_id"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	LastSeenVisible      bool   `json:"last_seen_visible"`
	ProfilePhotoVisible  bool   `json:"profile_photo_visible"`
	StatusVisible        bool   `json:"status_visible"`
	ReadReceiptsEnabled  bool   `json:"read_receipts_enabled"`
	UpdatedAt            int64  `json:"updated_at"`
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
