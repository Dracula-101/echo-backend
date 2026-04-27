package dto

import (
	"echo-backend/services/message-service/internal/domain"
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type MentionDTO struct {
	UserID   string `json:"user_id" validate:"required,uuid4"`
	Username string `json:"username,omitempty"`
	Offset   int    `json:"offset" validate:"gte=0"`
	Length   int    `json:"length" validate:"gt=0"`
}

type LocationDTO struct {
	Latitude  float64 `json:"latitude" validate:"gte=-90,lte=90"`
	Longitude float64 `json:"longitude" validate:"gte=-180,lte=180"`
	Name      string  `json:"name,omitempty" validate:"max=255"`
	Address   string  `json:"address,omitempty" validate:"max=500"`
}

type ContactDTO struct {
	Name  string `json:"name" validate:"required,max=100"`
	Phone string `json:"phone,omitempty" validate:"max=20"`
	Email string `json:"email,omitempty" validate:"omitempty,email,max=255"`
}

type SendMessageRequest struct {
	Content         string             `json:"content,omitempty" validate:"max=65536"`
	MessageType     domain.MessageType `json:"message_type" validate:"required,oneof=text image video audio document sticker gif location contact poll voice_note"`
	ParentMessageID *string            `json:"parent_message_id,omitempty" validate:"omitempty,uuid4"`
	ForwardedFrom   *string            `json:"forwarded_from,omitempty" validate:"omitempty,uuid4"`
	Mentions        []MentionDTO       `json:"mentions,omitempty" validate:"omitempty,max=50,dive"`
	MediaIDs        []string           `json:"media_ids,omitempty" validate:"omitempty,max=10,dive,uuid4"`
	StickerID       *string            `json:"sticker_id,omitempty" validate:"omitempty,uuid4"`
	GifID           *string            `json:"gif_id,omitempty" validate:"omitempty,uuid4"`
	Location        *LocationDTO       `json:"location,omitempty"`
	Contact         *ContactDTO        `json:"contact,omitempty"`
	ScheduledAt     *time.Time         `json:"scheduled_at,omitempty"`
	ExpireAfter     *int               `json:"expire_after,omitempty" validate:"omitempty,gt=0,lte=604800"`
}

func NewSendMessageRequest() *SendMessageRequest {
	return &SendMessageRequest{}
}

func (r *SendMessageRequest) GetValue() interface{} {
	return r
}

func (r *SendMessageRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	errors := make([]request.ValidationErrorDetail, 0, len(ve))
	for _, fieldErr := range ve {
		switch fieldErr.Field() {
		case "Content":
			if fieldErr.Tag() == "max" {
				errors = append(errors, request.ValidationErrorDetail{
					Code: request.TOO_LONG,
					Msg:  "Message content must be at most 65536 characters",
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
					Msg:  "Message type must be one of: text, image, video, audio, document, sticker, gif, location, contact, poll, voice_note",
				})
			}
		case "ParentMessageID", "ForwardedFrom", "StickerID", "GifID":
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  fieldErr.Field() + " must be a valid UUID",
			})
		case "Mentions":
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  "Mentions must be a list of at most 50 valid entries",
			})
		case "MediaIDs":
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  "media_ids must be a list of at most 10 valid UUIDs",
			})
		case "ExpireAfter":
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  "expire_after must be between 1 and 604800 seconds",
			})
		case "Latitude", "Longitude":
			errors = append(errors, request.ValidationErrorDetail{
				Code: request.INVALID_FORMAT,
				Msg:  fieldErr.Field() + " is out of range",
			})
		}
	}
	return errors, nil
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	MessageID      uuid.UUID            `json:"message_id"`
	ConversationID uuid.UUID            `json:"conversation_id"`
	SenderUserID   uuid.UUID            `json:"sender_user_id"`
	MessageType    domain.MessageType   `json:"message_type"`
	Content        string               `json:"content"`
	CreatedAt      time.Time            `json:"created_at"`
	ExpiresAt      *time.Time           `json:"expires_at,omitempty"`
	Status         domain.MessageStatus `json:"status"`
	DeliveryCount  int                  `json:"delivery_count"`
	ReadCount      int                  `json:"read_count"`
	IsScheduled    bool                 `json:"is_scheduled"`
}

func NewSendMessageResponse(message *domain.Message) *SendMessageResponse {
	return &SendMessageResponse{
		MessageID:      message.ID,
		ConversationID: message.ConversationID,
		SenderUserID:   message.SenderUserID,
		MessageType:    message.MessageType,
		Content:        message.Content,
		CreatedAt:      message.CreatedAt,
		ExpiresAt:      message.ExpiresAt,
		Status:         message.Status,
		DeliveryCount:  message.DeliveryCount,
		ReadCount:      message.ReadCount,
		IsScheduled:    message.IsScheduled,
	}
}
