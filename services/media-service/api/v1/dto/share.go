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
