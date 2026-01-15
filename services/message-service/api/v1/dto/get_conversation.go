package dto

import (
	"shared/server/request"

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
	ID               string  `json:"id"`
	ConversationType string  `json:"conversation_type"`
	Title            string  `json:"title,omitempty"`
	AvatarURL        *string `json:"avatar_url,omitempty"`
	IsEncrypted      bool    `json:"is_encrypted"`
	IsPublic         bool    `json:"is_public"`
	MemberCount      int     `json:"member_count"`
	UnreadCount      int     `json:"unread_count"`
	LastMessageAt    *int64  `json:"last_message_at,omitempty"`
	CreatedAt        int64   `json:"created_at"`
}

// GetConversationsResponse represents the response for listing conversations
type GetConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	Total         int                    `json:"total"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	HasMore       bool                   `json:"has_more"`
}
