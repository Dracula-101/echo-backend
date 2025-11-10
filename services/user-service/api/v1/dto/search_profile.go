package dto

// SearchUsersRequest represents a request to search for users
type SearchUsersRequest struct {
	Query  string `json:"query" validate:"required,min=2,max=100"`
	Limit  int    `json:"limit" validate:"omitempty,min=1,max=50"`
	Offset int    `json:"offset" validate:"omitempty,min=0"`
}

// SearchUsersResponse represents the response for searching users
type SearchUsersResponse struct {
	Users      []UserSearchResult `json:"users"`
	TotalCount int                `json:"total_count"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

// UserSearchResult represents a single user search result
type UserSearchResult struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	IsVerified  bool    `json:"is_verified"`
}
