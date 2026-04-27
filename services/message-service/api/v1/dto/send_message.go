package dto

import (
	"echo-backend/services/message-service/internal/domain"
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeText      MessageType = "text"
	MessageTypeImage     MessageType = "image"
	MessageTypeVideo     MessageType = "video"
	MessageTypeAudio     MessageType = "audio"
	MessageTypeDocument  MessageType = "document"
	MessageTypeSticker   MessageType = "sticker"
	MessageTypeGIF       MessageType = "gif"
	MessageTypeLocation  MessageType = "location"
	MessageTypeContact   MessageType = "contact"
	MessageTypePoll      MessageType = "poll"
	MessageTypeVoiceNote MessageType = "voice_note"
)

type Mention struct {
	UserID   uuid.UUID `json:"user_id" validate:"required,uuid4"`
	Username string    `json:"username" validate:"required"`
	Offset   int       `json:"offset" validate:"required,gte=0"`
	Length   int       `json:"length" validate:"required,gt=0"`
}

type Location struct {
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
	Name      string  `json:"name,omitempty" validate:"max=255"`
	Address   string  `json:"address,omitempty" validate:"max=500"`
}

type Contact struct {
	Name  string `json:"name" validate:"required"`
	Phone string `json:"phone,omitempty" validate:"omitempty,e164"`
	Email string `json:"email,omitempty" validate:"omitempty,email"`
}

type SendMessageRequest struct {
	MessageType     MessageType `json:"message_type" validate:"required,oneof=text image video audio document sticker gif location contact poll voice_note"`
	Content         string      `json:"content" validate:"required_if=MessageType text,omitempty,min=1,max=10000"`
	ParentMessageID *uuid.UUID  `json:"parent_message_id,omitempty" validate:"omitempty,uuid4"`
	ForwardedFrom   *uuid.UUID  `json:"forwarded_from,omitempty" validate:"omitempty,uuid4"`
	Mentions        []Mention   `json:"mentions,omitempty" validate:"max=50,dive"`
	StickerID       *uuid.UUID  `json:"sticker_id,omitempty" validate:"required_if=MessageType sticker,omitempty,uuid4"`
	GifID           *uuid.UUID  `json:"gif_id,omitempty" validate:"required_if=MessageType gif,omitempty,uuid4"`
	Location        *Location   `json:"location,omitempty" validate:"required_if=MessageType location,omitempty,dive"`
	Contact         *Contact    `json:"contact,omitempty" validate:"required_if=MessageType contact,omitempty,dive"`
	ScheduledAt     *time.Time  `json:"scheduled_at,omitempty" validate:"omitempty,gt"`
	ExpireAfter     *int        `json:"expire_after,omitempty" validate:"omitempty,gt=0"`
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
		case "MessageType":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Invalid message type. Allowed values are: text, image, video, audio, document, sticker, gif, location, contact, poll, voice_note.",
				Code: request.INVALID_FORMAT,
			})
		case "Content":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Content is required for text messages and must be between 1 and 10000 characters.",
				Code: request.REQUIRED_FIELD,
			})
		case "ParentMessageID", "ForwardedFrom", "StickerID", "GifID":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Must be a valid UUID.",
				Code: request.INVALID_FORMAT,
			})
		case "Mentions":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Mentions cannot exceed 50 and must be valid.",
				Code: request.INVALID_FORMAT,
			})
		case "Location":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Location must include valid latitude and longitude.",
				Code: request.INVALID_FORMAT,
			})
		case "Contact":
			errors = append(errors, request.ValidationErrorDetail{
				Msg:  "Contact must include a name and valid phone or email.",
				Code: request.INVALID_FORMAT,
			})
		case "ScheduledAt":
			if fieldErr.Tag() == "gt" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Scheduled time must be in the future.",
					Code: request.INVALID_FORMAT,
				})
			}
		case "ExpireAfter":
			if fieldErr.Tag() == "gt" {
				errors = append(errors, request.ValidationErrorDetail{
					Msg:  "Expire after must be a positive integer.",
					Code: request.INVALID_FORMAT,
				})
			}
		}
	}
	return errors, nil
}

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	ID              uuid.UUID     `json:"id"`
	ConversationID  uuid.UUID     `json:"conversation_id"`
	SenderUserID    uuid.UUID     `json:"sender_user_id"`
	ParentMessageID *uuid.UUID    `json:"parent_message_id,omitempty"`
	Content         string        `json:"content"`
	MessageType     MessageType   `json:"message_type"`
	Status          MessageStatus `json:"status"`
	IsEdited        bool          `json:"is_edited"`
	IsDeleted       bool          `json:"is_deleted"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	DeletedAt       *time.Time    `json:"deleted_at,omitempty"`
	EditedAt        *time.Time    `json:"edited_at,omitempty"`
}

func NewSendMessageResponse(message *domain.Message) *SendMessageResponse {
	return &SendMessageResponse{
		ID:              message.ID,
		ConversationID:  message.ConversationID,
		SenderUserID:    message.SenderUserID,
		ParentMessageID: message.ParentMessageID,
		Content:         message.Content,
		MessageType:     MessageType(message.MessageType),
		Status:          MessageStatus(message.Status),
		IsEdited:        message.IsEdited,
		IsDeleted:       message.IsDeleted,
		CreatedAt:       message.CreatedAt,
		UpdatedAt:       message.UpdatedAt,
		DeletedAt:       message.DeletedAt,
		EditedAt:        message.EditedAt,
	}
}
