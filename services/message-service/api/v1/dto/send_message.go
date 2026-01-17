package dto

import (
	"echo-backend/services/message-service/internal/domain"
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// SendMessageRequest represents the request to send a new message
type SendMessageRequest struct {
	ConversationID  string                  `json:"conversation_id" validate:"required,uuid4"`
	Content         string                  `json:"content" validate:"required,min=1,max=10000"`
	MessageType     domain.MessageType      `json:"message_type" validate:"required,oneof=text image video audio document location contact poll"`
	ParentMessageID *string                 `json:"parent_message_id,omitempty" validate:"omitempty,uuid4"`
	Mentions        []domain.Mention        `json:"mentions,omitempty"`
	Metadata        *domain.MessageMetadata `json:"metadata,omitempty"`
}

func NewSendMessageRequest() *SendMessageRequest {
	return &SendMessageRequest{}
}

func (r *SendMessageRequest) GetValue() interface{} {
	return r
}

func (r *SendMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationID":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Conversation ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Conversation ID must be a valid UUID",
				})
			}
		case "Content":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Message content is required",
				})
			} else if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_SHORT,
					Msg:  "Message content cannot be empty",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Message content must be at most 10000 characters",
				})
			}
		case "MessageType":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Message type is required",
				})
			} else if fieldErr.Tag() == "oneof" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Message type must be one of: text, image, video, audio, document, location, contact, poll",
				})
			}
		case "ParentMessageID":
			if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Parent message ID must be a valid UUID",
				})
			}
		}
	}
	return errors, nil
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	ID              uuid.UUID               `json:"id"`
	ConversationID  uuid.UUID               `json:"conversation_id"`
	SenderUserID    uuid.UUID               `json:"sender_user_id"`
	ParentMessageID *uuid.UUID              `json:"parent_message_id,omitempty"`
	Content         string                  `json:"content"`
	MessageType     domain.MessageType      `json:"message_type"`
	Status          domain.MessageStatus    `json:"status"`
	IsEdited        bool                    `json:"is_edited"`
	IsDeleted       bool                    `json:"is_deleted"`
	Mentions        []domain.Mention        `json:"mentions"`
	Metadata        *domain.MessageMetadata `json:"metadata"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	DeletedAt       *time.Time              `json:"deleted_at,omitempty"`
	EditedAt        *time.Time              `json:"edited_at,omitempty"`
}

func NewSendMessageResponse(message *domain.Message) *SendMessageResponse {
	return &SendMessageResponse{
		ID:              message.ID,
		ConversationID:  message.ConversationID,
		SenderUserID:    message.SenderUserID,
		ParentMessageID: message.ParentMessageID,
		Content:         message.Content,
		MessageType:     message.MessageType,
		Status:          message.Status,
		IsEdited:        message.IsEdited,
		IsDeleted:       message.IsDeleted,
		Mentions:        message.Mentions,
		Metadata:        message.Metadata,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
		DeletedAt:       message.DeletedAt,
		EditedAt:        message.EditedAt,
	}
}
