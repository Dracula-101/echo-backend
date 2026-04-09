package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// GetConversationsRequest represents the request to list conversations
type GetConversationsRequest struct {
	Limit  int `json:"limit" validate:"omitempty,min=1,max=100"`
	Offset int `json:"offset" validate:"omitempty,min=0"`
}

func NewGetConversationsRequest() *GetConversationsRequest {
	return &GetConversationsRequest{
		Limit:  20,
		Offset: 0,
	}
}

func (r *GetConversationsRequest) GetValue() interface{} {
	return r
}

func (r *GetConversationsRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "Limit":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Limit must be at least 1",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Limit must be at most 100",
				})
			}
		case "Offset":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Offset must be at least 0",
				})
			}
		}
	}
	return errors, nil
}

// ConversationResponse represents a single conversation in the list
type ConversationResponse struct {
	ID               string        `json:"id"`
	Title            *string       `json:"title,omitempty"`
	Description      *string       `json:"description,omitempty"`
	ConversationType string        `json:"conversation_type"`
	AvatarURL        *string       `json:"avatar_url,omitempty"`
	UnreadCount      int           `json:"unread_count"`
	MemberCount      int           `json:"member_count"`
	MessageCount     int           `json:"message_count"`
	LastMessage      *Message      `json:"last_message,omitempty"`
	Participants     []Participant `json:"participants"`
}

type Message struct {
	Content      string `json:"content"`
	MessageType  string `json:"message_type"`
	SenderUserID string `json:"sender_user_id"`
	SenderName   string `json:"sender_name,omitempty"`
	SenderAvatar string `json:"sender_avatar,omitempty"`
}

type Participant struct {
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name,omitempty"`
	DisplayName  string    `json:"display_name,omitempty"`
	UserAvatar   string    `json:"user_avatar,omitempty"`
	FirstName    string    `json:"first_name,omitempty"`
	LastName     string    `json:"last_name,omitempty"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	LastSeen     *time.Time `json:"last_seen,omitempty"`
	OnlineStatus string    `json:"online_status,omitempty"`
	IsVerified   bool      `json:"is_verified,omitempty"`
}

type GetConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	Total         int                    `json:"total"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	HasMore       bool                   `json:"has_more"`
}
