package dto

import (
	"echo-backend/services/message-service/internal/domain"
	"time"
)

// MessageResponse represents a single message in the response
type MessageResponse struct {
	ID              string                  `json:"id"`
	ConversationID  string                  `json:"conversation_id"`
	SenderUserID    string                  `json:"sender_user_id"`
	ParentMessageID *string                 `json:"parent_message_id,omitempty"`
	Content         string                  `json:"content"`
	MessageType     string                  `json:"message_type"`
	Status          string                  `json:"status"`
	IsEdited        bool                    `json:"is_edited"`
	IsDeleted       bool                    `json:"is_deleted"`
	ReadCount       int                     `json:"read_count"`
	Metadata        *domain.MessageMetadata `json:"metadata,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	EditedAt        *time.Time              `json:"edited_at,omitempty"`
	DeletedAt       *time.Time              `json:"deleted_at,omitempty"`
}

// GetMessagesResponse represents the paginated response for getting messages
type GetMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	HasMore  bool              `json:"has_more"`
	Limit    int               `json:"limit"`
}
