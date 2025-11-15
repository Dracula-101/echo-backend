package dto

import (
	"echo-backend/services/message-service/internal/models"
	"shared/server/request"

	"github.com/go-playground/validator/v10"
)

type GetMessagesRequest struct {
	ConversationID string `json:"conversation_id" validate:"required,uuid4"`
	Limit          int    `json:"limit" validate:"omitempty,min=1,max=100"`
	Offset         int    `json:"offset" validate:"omitempty,min=0"`
}

func NewGetMessagesRequest() *GetMessagesRequest {
	return &GetMessagesRequest{
		Limit:  20,
		Offset: 0,
	}
}

func (r *GetMessagesRequest) GetValue() interface{} {
	return r
}

func (r *GetMessagesRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "ConversationID":
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "conversation_id_required",
					Msg:  "Conversation ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_conversation_id",
					Msg:  "Conversation ID must be a valid UUIDv4",
				})
			}
		case "Limit":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_limit",
					Msg:  "Limit must be at least 1",
				})
			} else if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_limit",
					Msg:  "Limit must be at most 100",
				})
			}
		case "Offset":
			if fieldErr.Tag() == "min" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: "invalid_offset",
					Msg:  "Offset must be at least 0",
				})
			}
		default:
			errors = append(errors, request.ValidationErrorDetail{
				Code: fieldErr.Field(),
				Msg:  "Invalid value",
			})
		}
	}
	return errors, nil
}

type GetMessagesResponse struct {
	models.MessagesResponse
}

// SendMessageRequest represents the request to send a new message
type SendMessageRequest struct {
	ConversationID  string                 `json:"conversation_id" validate:"required,uuid4"`
	Content         string                 `json:"content" validate:"required,min=1,max=10000"`
	MessageType     string                 `json:"message_type" validate:"required,oneof=text image video audio document location contact poll"`
	ParentMessageID *string                `json:"parent_message_id,omitempty" validate:"omitempty,uuid4"`
	Mentions        []models.Mention       `json:"mentions,omitempty"`
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
	Mentions        []models.Mention       `json:"mentions,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
}

func NewSendMessageResponse(msg *models.Message) *SendMessageResponse {
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
		CreatedAt:       msg.CreatedAt.Unix(),
		UpdatedAt:       msg.UpdatedAt.Unix(),
	}
}

// EditMessageRequest represents the request to edit a message
type EditMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

func NewEditMessageRequest() *EditMessageRequest {
	return &EditMessageRequest{}
}

func (r *EditMessageRequest) GetValue() interface{} {
	return r
}

func (r *EditMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "Content" {
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
		}
	}
	return errors, nil
}

// MarkAsReadRequest represents the request to mark a message as read
type MarkAsReadRequest struct {
	MessageID string `json:"message_id" validate:"required,uuid4"`
}

func NewMarkAsReadRequest() *MarkAsReadRequest {
	return &MarkAsReadRequest{}
}

func (r *MarkAsReadRequest) GetValue() interface{} {
	return r
}

func (r *MarkAsReadRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "MessageID" {
			if fieldErr.Tag() == "required" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.REQUIRED_FIELD,
					Msg:  "Message ID is required",
				})
			} else if fieldErr.Tag() == "uuid4" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.INVALID_FORMAT,
					Msg:  "Message ID must be a valid UUID",
				})
			}
		}
	}
	return errors, nil
}

// TypingIndicatorRequest represents the request to set typing indicator
type TypingIndicatorRequest struct {
	ConversationID string `json:"conversation_id" validate:"required,uuid4"`
	IsTyping       bool   `json:"is_typing"`
}

func NewTypingIndicatorRequest() *TypingIndicatorRequest {
	return &TypingIndicatorRequest{}
}

func (r *TypingIndicatorRequest) GetValue() interface{} {
	return r
}

func (r *TypingIndicatorRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	for _, fieldErr := range ve {
		if fieldErr.Field() == "ConversationID" {
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
		}
	}
	return errors, nil
}
