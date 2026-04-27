package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type UpdateConversationRequest struct {
	Title                *string `json:"title,omitempty" validate:"omitempty,max=255"`
	Description          *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	AvatarURL            *string `json:"avatar_url,omitempty"`
	AvatarFileID         *string `json:"avatar_file_id,omitempty" validate:"omitempty,uuid4"`
	IsPublic             *bool   `json:"is_public,omitempty"`
	WhoCanSendMessages   *string `json:"who_can_send,omitempty" validate:"omitempty,oneof=all admins"`
	WhoCanAddMembers     *string `json:"who_can_add,omitempty" validate:"omitempty,oneof=all admins"`
	WhoCanEditInfo       *string `json:"who_can_edit,omitempty" validate:"omitempty,oneof=all admins"`
	WhoCanPinMessages    *string `json:"who_can_pin,omitempty" validate:"omitempty,oneof=all admins"`
	InviteLinkExpiresAt  *string `json:"invite_link_expires_at,omitempty"`
	JoinApprovalRequired *bool   `json:"join_approval_required,omitempty"`
}

func NewUpdateConversationRequest() *UpdateConversationRequest {
	return &UpdateConversationRequest{}
}

func (r *UpdateConversationRequest) GetValue() interface{} {
	return r
}

func (r *UpdateConversationRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "Title":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "Title must be at most 255 characters"})
			}
		case "Description":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.TOO_LONG, Msg: "Description must be at most 1000 characters"})
			}
		case "AvatarFileID":
			if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.INVALID_FORMAT, Msg: "avatar_file_id must be a valid UUID"})
			}
		case "WhoCanSendMessages", "WhoCanAddMembers", "WhoCanEditInfo", "WhoCanPinMessages":
			if fieldErr.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{Code: request.INVALID_FORMAT, Msg: "who_can_* must be 'all' or 'admins'"})
			}
		}
	}
	return errors, nil
}
