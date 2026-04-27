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
	IsScheduled     bool
	ExpiresAt       *time.Time
	Metadata        *MessageMetadata
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	EditedAt        *time.Time

	SenderName    string
	SenderAvatar  string
	ReadCount     int
	DeliveryCount int
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
		if err := json.Unmarshal(model.Metadata, &metadata); err == nil {
			msg.Metadata = &MessageMetadata{
				MediaURL:      getString(metadata, "media_url"),
				ThumbnailURL:  getString(metadata, "thumbnail_url"),
				MediaSize:     getInt(metadata, "media_size"),
				MediaDuration: getInt(metadata, "media_duration"),
				MimeType:      getString(metadata, "mime_type"),
				FileName:      getString(metadata, "file_name"),
				Width:         getInt(metadata, "width"),
				Height:        getInt(metadata, "height"),
				Latitude:      getFloat(metadata, "latitude"),
				Longitude:     getFloat(metadata, "longitude"),
				Address:       getString(metadata, "address"),
			}
			if lp, ok := metadata["link_preview"].(map[string]interface{}); ok {
				msg.Metadata.LinkPreview = &LinkPreview{
					URL:         getString(lp, "url"),
					Title:       getString(lp, "title"),
					Description: getString(lp, "description"),
					Image:       getString(lp, "image"),
					SiteName:    getString(lp, "site_name"),
				}
			}
		}
	}

	return msg
}

// MessageType represents the type of message content
type MessageType string

const (
	MessageTypeText      MessageType = "text"
	MessageTypeImage     MessageType = "image"
	MessageTypeVideo     MessageType = "video"
	MessageTypeAudio     MessageType = "audio"
	MessageTypeFile      MessageType = "file"
	MessageTypeDocument  MessageType = "document"
	MessageTypeLocation  MessageType = "location"
	MessageTypeContact   MessageType = "contact"
	MessageTypeSticker   MessageType = "sticker"
	MessageTypeGif       MessageType = "gif"
	MessageTypePoll      MessageType = "poll"
	MessageTypeVoiceNote MessageType = "voice_note"
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

type MessageMetadata struct {
	MediaURL      string `json:"media_url,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	MediaSize     int    `json:"media_size,omitempty"`
	MediaDuration int    `json:"media_duration,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`

	MediaIDs []string `json:"media_ids,omitempty"`

	StickerID string `json:"sticker_id,omitempty"`
	GifID     string `json:"gif_id,omitempty"`

	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	LocationName string  `json:"location_name,omitempty"`
	Address      string  `json:"address,omitempty"`

	ContactName  string `json:"contact_name,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`

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

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}
