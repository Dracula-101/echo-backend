package domain

import (
	"time"

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
	Mentions        []uuid.UUID
	Metadata        MessageMetadata
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	EditedAt        *time.Time

	// Enriched fields (from joins)
	SenderName   string
	SenderAvatar string
	ReadCount    int
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
	MediaURL      string  `json:"media_url,omitempty"`
	ThumbnailURL  string  `json:"thumbnail_url,omitempty"`
	MediaSize     int64   `json:"media_size,omitempty"`
	MediaDuration int     `json:"media_duration,omitempty"`
	MimeType      string  `json:"mime_type,omitempty"`
	FileName      string  `json:"file_name,omitempty"`
	Width         int     `json:"width,omitempty"`
	Height        int     `json:"height,omitempty"`

	// Location information
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Address   string  `json:"address,omitempty"`

	// Other metadata
	LinkPreview *LinkPreview `json:"link_preview,omitempty"`
}

// LinkPreview contains URL preview information
type LinkPreview struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
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

// DeliveryStatus represents message delivery status per recipient
type DeliveryStatus struct {
	ID          uuid.UUID
	MessageID   uuid.UUID
	RecipientID uuid.UUID
	Status      MessageStatus
	DeliveredAt *time.Time
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsRead checks if message has been read
func (d *DeliveryStatus) IsRead() bool {
	return d.Status == MessageStatusRead && d.ReadAt != nil
}

// IsDelivered checks if message has been delivered
func (d *DeliveryStatus) IsDelivered() bool {
	return d.Status == MessageStatusDelivered || d.Status == MessageStatusRead
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
