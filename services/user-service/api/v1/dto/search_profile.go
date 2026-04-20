package dto

type UserSearchResult struct {
	UserID             string  `json:"user_id"`
	DisplayName        string  `json:"display_name"`
	Username           string  `json:"username"`
	AvatarURL          *string `json:"avatar_url,omitempty"`
	AvatarThumbnailURL *string `json:"avatar_thumbnail_url,omitempty"`
	Bio                *string `json:"bio,omitempty"`
	IsVerified         bool    `json:"is_verified"`
	IsContact          bool    `json:"is_contact"`
	ProfileVisibility  string  `json:"profile_visibility"`
}

type SearchUsersResponse struct {
	Users      []UserSearchResult `json:"users"`
	TotalCount int                `json:"total_count"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}
