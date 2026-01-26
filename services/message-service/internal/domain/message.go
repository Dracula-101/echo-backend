package domain

import (
	"encoding/json"
	"shared/pkg/utils"
	"time"

	db "shared/pkg/database/postgres/models"

	"github.com/google/uuid"
)

// Message represents the core message entity
type Message struct {
	ID              uuid.UUID
	ConversationID  uuid.UUID
	SenderUserID    uuid.UUID
	ParentMessageID *uuid.UUID
	Content         string
	MessageType     MessageType
	Status          MessageStatus
	IsEdited        bool
	IsDeleted       bool
	Metadata        *MessageMetadata
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	EditedAt        *time.Time

	// Enriched fields (from joins)
	SenderName   string
	SenderAvatar string
	ReadCount    int
}

func NewMessage(model db.Message) Message {
	msg := Message{
		ID:              *utils.ParsePtrUUID(model.ID),
		ConversationID:  uuid.MustParse(model.ConversationID),
		SenderUserID:    uuid.MustParse(model.SenderUserID),
		ParentMessageID: utils.SafeParsePtrUUID(model.ParentMessageID),
		Content:         utils.DerefString(model.Content),
		MessageType:     MessageType(model.MessageType),
		Status:          MessageStatus(model.Status),
		IsEdited:        model.IsEdited,
		IsDeleted:       model.IsDeleted,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
		DeletedAt:       model.DeletedAt,
		EditedAt:        model.EditedAt,
		ReadCount:       int(model.ReadCount),
	}

	if model.Metadata != nil {
		metadata := make(map[string]interface{})
		_ = json.Unmarshal(model.Metadata, &metadata)
		msg.Metadata = &MessageMetadata{
			MediaURL:      utils.GetValue(metadata, "media_url", "").(string),
			ThumbnailURL:  utils.GetValue(metadata, "thumbnail_url", "").(string),
			MediaSize:     utils.GetValue(metadata, "media_size", 0).(int64),
			MediaDuration: utils.GetValue(metadata, "media_duration", 0).(int),
			MimeType:      utils.GetValue(metadata, "mime_type", "").(string),
			FileName:      utils.GetValue(metadata, "file_name", "").(string),
			Width:         utils.GetValue(metadata, "width", 0).(int),
			Height:        utils.GetValue(metadata, "height", 0).(int),
			Latitude:      utils.GetValue(metadata, "latitude", 0.0).(float64),
			Longitude:     utils.GetValue(metadata, "longitude", 0.0).(float64),
			Address:       utils.GetValue(metadata, "address", "").(string),
		}
		if lp, ok := metadata["link_preview"].(map[string]interface{}); ok {
			msg.Metadata.LinkPreview = &LinkPreview{
				URL:         utils.GetValue(lp, "url", "").(string),
				Title:       utils.GetValue(lp, "title", "").(string),
				Description: utils.GetValue(lp, "description", "").(string),
				Image:       utils.GetValue(lp, "image", "").(string),
				SiteName:    utils.GetValue(lp, "site_name", "").(string),
			}
		}
	}

	return msg
}

// MessageType represents the type of message content
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeFile     MessageType = "file"
	MessageTypeLocation MessageType = "location"
)

// MessageStatus represents the delivery status
type MessageStatus string

const (
	MessageStatusSending   MessageStatus = "sending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// MessageMetadata contains additional message information
type MessageMetadata struct {
	// Media information
	MediaURL      string `json:"media_url,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	MediaSize     int64  `json:"media_size,omitempty"`
	MediaDuration int    `json:"media_duration,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`

	// Location information
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Address   string  `json:"address,omitempty"`

	// Other metadata
	LinkPreview *LinkPreview `json:"link_preview,omitempty"`
}

// CanBeEditedBy checks if a message can be edited by the given user
func (m *Message) CanBeEditedBy(userID uuid.UUID) bool {
	if m.IsDeleted {
		return false
	}
	if m.SenderUserID != userID {
		return false
	}
	// Allow editing within 24 hours
	editDeadline := m.CreatedAt.Add(24 * time.Hour)
	return time.Now().Before(editDeadline)
}

// CanBeDeletedBy checks if a message can be deleted by the given user
func (m *Message) CanBeDeletedBy(userID uuid.UUID) bool {
	if m.IsDeleted {
		return false
	}
	return m.SenderUserID == userID
}

// IsExpired checks if message is older than retention period
func (m *Message) IsExpired(retentionDays int) bool {
	if retentionDays <= 0 {
		return false
	}
	expiryDate := m.CreatedAt.AddDate(0, 0, retentionDays)
	return time.Now().After(expiryDate)
}

// HasMedia checks if message contains media
func (m *Message) HasMedia() bool {
	return m.MessageType != MessageTypeText
}

// TypingIndicator represents a user typing in a conversation
type TypingIndicator struct {
	UserID         uuid.UUID
	ConversationID uuid.UUID
	StartedAt      time.Time
	ExpiresAt      time.Time
}

// IsActive checks if typing indicator is still active
func (t *TypingIndicator) IsActive() bool {
	return time.Now().Before(t.ExpiresAt)
}
