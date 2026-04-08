package dto

type CreateShareRequest struct {
	FileID         string `json:"file_id" validate:"required"`
	SharedWithUser string `json:"shared_with_user_id"`
	ConversationID string `json:"conversation_id"`
	AccessType     string `json:"access_type" validate:"required"`
	ExpiresInHours int    `json:"expires_in_hours"`
	MaxViews       int    `json:"max_views"`
	Password       string `json:"password"`
}

func NewCreateShareRequest() *CreateShareRequest {
	return &CreateShareRequest{}
}

type GetShareResponse struct {
	ShareID        string `json:"share_id"`
	FileID         string `json:"file_id"`
	SharedWithUser string `json:"shared_with_user_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	AccessType     string `json:"access_type"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	MaxViews       int    `json:"max_views,omitempty"`
	ViewCount      int    `json:"view_count"`
}

type ListSharesRequest struct {
	FileID    string `json:"file_id,omitempty"`
	Limit     int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset    int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	SortBy    string `json:"sort_by,omitempty" validate:"omitempty,oneof=created_at expires_at"`
	SortOrder string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListSharesResponse struct {
	Shares     []GetShareResponse `json:"shares"`
	TotalCount int                `json:"total_count"`
	HasMore    bool               `json:"has_more"`
}
