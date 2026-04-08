package dto

import (
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// MuteConversationRequest represents the request to mute a conversation
type MuteConversationRequest struct {
	MuteUntil *string `json:"mute_until,omitempty"`
}

func NewMuteConversationRequest() *MuteConversationRequest {
	return &MuteConversationRequest{}
}

func (r *MuteConversationRequest) GetValue() interface{} {
	return r
}

func (r *MuteConversationRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	return nil, nil
}
