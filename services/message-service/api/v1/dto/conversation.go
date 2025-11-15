package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// CreateConversationRequest represents the request to create a new conversation
type CreateConversationRequest struct {
	ConversationType string   `json:"conversation_type" validate:"required,oneof=direct group channel broadcast"`
	ParticipantIDs   []string `json:"participant_ids" validate:"required,min=1,dive,uuid4"`
	Title            string   `json:"title,omitempty" validate:"omitempty,max=255"`
	Description      string   `json:"description,omitempty" validate:"omitempty,max=1000"`
	IsEncrypted      bool     `json:"is_encrypted"`
	IsPublic         bool     `json:"is_public"`
}

func NewCreateConversationRequest() *CreateConversationRequest {
	return &CreateConversationRequest{
		IsEncrypted: true,
		IsPublic:    false,
	}
}

func (r *CreateConversationRequest) GetValue() interface{} {
	return r
}

func (r *CreateConversationRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationType":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Conversation type is required",
				})
			} else if fieldErr.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Conversation type must be one of: direct, group, channel, broadcast",
				})
			}
		case "ParticipantIDs":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Participant IDs are required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "At least one participant is required",
				})
			}
		case "Title":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Title must be at most 255 characters",
				})
			}
		case "Description":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Description must be at most 1000 characters",
				})
			}
		}
	}
	return errors, nil
}

// CreateConversationResponse represents the response after creating a conversation
type CreateConversationResponse struct {
	ID               string   `json:"id"`
	ConversationType string   `json:"conversation_type"`
	Title            string   `json:"title,omitempty"`
	Description      string   `json:"description,omitempty"`
	CreatorUserID    string   `json:"creator_user_id"`
	IsEncrypted      bool     `json:"is_encrypted"`
	IsPublic         bool     `json:"is_public"`
	MemberCount      int      `json:"member_count"`
	ParticipantIDs   []string `json:"participant_ids"`
	CreatedAt        int64    `json:"created_at"`
}

func NewCreateConversationResponse(
	id uuid.UUID,
	conversationType string,
	title string,
	description string,
	creatorUserID uuid.UUID,
	isEncrypted bool,
	isPublic bool,
	participantIDs []uuid.UUID,
	createdAt int64,
) *CreateConversationResponse {
	pIDs := make([]string, len(participantIDs))
	for i, pid := range participantIDs {
		pIDs[i] = pid.String()
	}

	return &CreateConversationResponse{
		ID:               id.String(),
		ConversationType: conversationType,
		Title:            title,
		Description:      description,
		CreatorUserID:    creatorUserID.String(),
		IsEncrypted:      isEncrypted,
		IsPublic:         isPublic,
		MemberCount:      len(participantIDs),
		ParticipantIDs:   pIDs,
		CreatedAt:        createdAt,
	}
}

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
