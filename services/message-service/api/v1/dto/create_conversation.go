package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// CreateConversationRequest represents the request to create a new conversation
type CreateConversationRequest struct {
	ConversationType     string   `json:"conversation_type" validate:"required,oneof=direct group channel broadcast"`
	ParticipantIDs       []string `json:"participant_ids" validate:"required,min=1,dive,uuid4"`
	Title                string   `json:"title,omitempty" validate:"omitempty,max=255"`
	Description          string   `json:"description,omitempty" validate:"omitempty,max=1000"`
	MaxMembers           *int     `json:"max_members,omitempty" validate:"omitempty,gt=0"`
	JoinApprovalRequired bool     `json:"join_approval_required"`
	WhoCanSendMessages   string   `json:"who_can_send_messages,omitempty" validate:"omitempty,oneof=all admins only"`
	WhoCanAddMembers     string   `json:"who_can_add_members,omitempty" validate:"omitempty,oneof=all admins only"`
	WhoCanEditInfo       string   `json:"who_can_edit_info,omitempty" validate:"omitempty,oneof=all admins only"`
	WhoCanPinMessages    string   `json:"who_can_pin_messages,omitempty" validate:"omitempty,oneof=all admins only"`
	IsEncrypted          bool     `json:"is_encrypted"`
	AvatarURL            *string  `json:"avatar_url,omitempty"`
	IsPublic             bool     `json:"is_public"`
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
	AvatarURL        *string  `json:"avatar_url,omitempty"`
	MemberCount      int      `json:"member_count"`
	ParticipantIDs   []string `json:"participant_ids"`
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
	}
}
