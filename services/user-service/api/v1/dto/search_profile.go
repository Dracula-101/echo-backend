package dto

type UserSearchResult struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Username    string  `json:"username"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type SearchUsersResponse struct {
	Users      []UserSearchResult `json:"users"`
	TotalCount int                `json:"total_count"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}
