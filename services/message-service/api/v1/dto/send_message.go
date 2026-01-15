package dto

import (
	svcmodel "echo-backend/services/message-service/internal/service/model"
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

// SendMessageRequest represents the request to send a new message
type SendMessageRequest struct {
	ConversationID  string                 `json:"conversation_id" validate:"required,uuid4"`
	Content         string                 `json:"content" validate:"required,min=1,max=10000"`
	MessageType     string                 `json:"message_type" validate:"required,oneof=text image video audio document location contact poll"`
	ParentMessageID *string                `json:"parent_message_id,omitempty" validate:"omitempty,uuid4"`
	Mentions        []svcmodel.Mention     `json:"mentions,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
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
	ID              string                 `json:"id"`
	ConversationID  string                 `json:"conversation_id"`
	SenderUserID    string                 `json:"sender_user_id"`
	Content         string                 `json:"content"`
	MessageType     string                 `json:"message_type"`
	Status          string                 `json:"status"`
	ParentMessageID *string                `json:"parent_message_id,omitempty"`
	Mentions        []svcmodel.Mention     `json:"mentions,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
}

func NewSendMessageResponse(msg *svcmodel.Message) *SendMessageResponse {
	var parentMsgID *string
	if msg.ParentMessageID != nil {
		id := msg.ParentMessageID.String()
		parentMsgID = &id
	}

	return &SendMessageResponse{
		ID:              msg.ID.String(),
		ConversationID:  msg.ConversationID.String(),
		SenderUserID:    msg.SenderUserID.String(),
		Content:         msg.Content,
		MessageType:     msg.MessageType,
		Status:          msg.Status,
		ParentMessageID: parentMsgID,
		Mentions:        msg.Mentions,
		Metadata:        map[string]interface{}(msg.Metadata),
		CreatedAt:       msg.CreatedAt.Unix(),
		UpdatedAt:       msg.UpdatedAt.Unix(),
	}
}
