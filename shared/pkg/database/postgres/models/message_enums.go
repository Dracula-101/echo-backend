package models

import (
	"database/sql/driver"
	"fmt"
)

// ConversationType represents the type of conversation
type ConversationType string

const (
	ConversationTypeDirect    ConversationType = "direct"
	ConversationTypeGroup     ConversationType = "group"
	ConversationTypeChannel   ConversationType = "channel"
	ConversationTypeBroadcast ConversationType = "broadcast"
)

func (c ConversationType) IsValid() bool {
	switch c {
	case ConversationTypeDirect, ConversationTypeGroup, ConversationTypeChannel, ConversationTypeBroadcast:
		return true
	}
	return false
}

func (c ConversationType) Value() (driver.Value, error) {
	if !c.IsValid() {
		return nil, fmt.Errorf("invalid conversation type: %s", c)
	}
	return string(c), nil
}

func (c *ConversationType) Scan(value interface{}) error {
	if value == nil {
		*c = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ConversationType: expected string, got %T", value)
	}
	*c = ConversationType(str)
	if !c.IsValid() {
		return fmt.Errorf("invalid conversation type value: %s", str)
	}
	return nil
}

// ParticipantRole represents the role of a participant in a conversation
type ParticipantRole string

const (
	ParticipantRoleMember    ParticipantRole = "member"
	ParticipantRoleAdmin     ParticipantRole = "admin"
	ParticipantRoleModerator ParticipantRole = "moderator"
	ParticipantRoleOwner     ParticipantRole = "owner"
)

func (p ParticipantRole) IsValid() bool {
	switch p {
	case ParticipantRoleMember, ParticipantRoleAdmin, ParticipantRoleModerator, ParticipantRoleOwner:
		return true
	}
	return false
}

func (p ParticipantRole) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid participant role: %s", p)
	}
	return string(p), nil
}

func (p *ParticipantRole) Scan(value interface{}) error {
	if value == nil {
		*p = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ParticipantRole: expected string, got %T", value)
	}
	*p = ParticipantRole(str)
	if !p.IsValid() {
		return fmt.Errorf("invalid participant role value: %s", str)
	}
	return nil
}

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeDocument MessageType = "document"
	MessageTypeLocation MessageType = "location"
	MessageTypeContact  MessageType = "contact"
	MessageTypeSticker  MessageType = "sticker"
	MessageTypePoll     MessageType = "poll"
	MessageTypeSystem   MessageType = "system"
)

func (m MessageType) IsValid() bool {
	switch m {
	case MessageTypeText, MessageTypeImage, MessageTypeVideo, MessageTypeAudio,
		MessageTypeDocument, MessageTypeLocation, MessageTypeContact,
		MessageTypeSticker, MessageTypePoll, MessageTypeSystem:
		return true
	}
	return false
}

func (m MessageType) Value() (driver.Value, error) {
	if !m.IsValid() {
		return nil, fmt.Errorf("invalid message type: %s", m)
	}
	return string(m), nil
}

func (m *MessageType) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan MessageType: expected string, got %T", value)
	}
	*m = MessageType(str)
	if !m.IsValid() {
		return fmt.Errorf("invalid message type value: %s", str)
	}
	return nil
}

// MessageFormatType represents the format of message content
type MessageFormatType string

const (
	MessageFormatPlain    MessageFormatType = "plain"
	MessageFormatMarkdown MessageFormatType = "markdown"
	MessageFormatHTML     MessageFormatType = "html"
)

func (m MessageFormatType) IsValid() bool {
	switch m {
	case MessageFormatPlain, MessageFormatMarkdown, MessageFormatHTML:
		return true
	}
	return false
}

func (m MessageFormatType) Value() (driver.Value, error) {
	if !m.IsValid() {
		return nil, fmt.Errorf("invalid message format type: %s", m)
	}
	return string(m), nil
}

func (m *MessageFormatType) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan MessageFormatType: expected string, got %T", value)
	}
	*m = MessageFormatType(str)
	if !m.IsValid() {
		return fmt.Errorf("invalid message format type value: %s", str)
	}
	return nil
}

// MessageStatus represents the status of a message
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

func (m MessageStatus) IsValid() bool {
	switch m {
	case MessageStatusPending, MessageStatusSent, MessageStatusDelivered, MessageStatusRead, MessageStatusFailed:
		return true
	}
	return false
}

func (m MessageStatus) Value() (driver.Value, error) {
	if !m.IsValid() {
		return nil, fmt.Errorf("invalid message status: %s", m)
	}
	return string(m), nil
}

func (m *MessageStatus) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan MessageStatus: expected string, got %T", value)
	}
	*m = MessageStatus(str)
	if !m.IsValid() {
		return fmt.Errorf("invalid message status value: %s", str)
	}
	return nil
}

// DeliveryStatusType represents the delivery status for a specific user
type DeliveryStatusType string

const (
	DeliveryStatusPending   DeliveryStatusType = "pending"
	DeliveryStatusDelivered DeliveryStatusType = "delivered"
	DeliveryStatusRead      DeliveryStatusType = "read"
	DeliveryStatusFailed    DeliveryStatusType = "failed"
)

func (d DeliveryStatusType) IsValid() bool {
	switch d {
	case DeliveryStatusPending, DeliveryStatusDelivered, DeliveryStatusRead, DeliveryStatusFailed:
		return true
	}
	return false
}

func (d DeliveryStatusType) Value() (driver.Value, error) {
	if !d.IsValid() {
		return nil, fmt.Errorf("invalid delivery status: %s", d)
	}
	return string(d), nil
}

func (d *DeliveryStatusType) Scan(value interface{}) error {
	if value == nil {
		*d = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan DeliveryStatusType: expected string, got %T", value)
	}
	*d = DeliveryStatusType(str)
	if !d.IsValid() {
		return fmt.Errorf("invalid delivery status value: %s", str)
	}
	return nil
}

// MediaType represents the type of media
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
	MediaTypeFile     MediaType = "file"
)

func (m MediaType) IsValid() bool {
	switch m {
	case MediaTypeImage, MediaTypeVideo, MediaTypeAudio, MediaTypeDocument, MediaTypeFile:
		return true
	}
	return false
}

func (m MediaType) Value() (driver.Value, error) {
	if !m.IsValid() {
		return nil, fmt.Errorf("invalid media type: %s", m)
	}
	return string(m), nil
}

func (m *MediaType) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan MediaType: expected string, got %T", value)
	}
	*m = MediaType(str)
	if !m.IsValid() {
		return fmt.Errorf("invalid media type value: %s", str)
	}
	return nil
}

// ReportStatus represents the status of a report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusInReview  ReportStatus = "in_review"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

func (r ReportStatus) IsValid() bool {
	switch r {
	case ReportStatusPending, ReportStatusInReview, ReportStatusResolved, ReportStatusDismissed:
		return true
	}
	return false
}

func (r ReportStatus) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid report status: %s", r)
	}
	return string(r), nil
}

func (r *ReportStatus) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ReportStatus: expected string, got %T", value)
	}
	*r = ReportStatus(str)
	if !r.IsValid() {
		return fmt.Errorf("invalid report status value: %s", str)
	}
	return nil
}

// ReportPriority represents the priority level of a report
type ReportPriority string

const (
	ReportPriorityLow      ReportPriority = "low"
	ReportPriorityMedium   ReportPriority = "medium"
	ReportPriorityHigh     ReportPriority = "high"
	ReportPriorityCritical ReportPriority = "critical"
)

func (r ReportPriority) IsValid() bool {
	switch r {
	case ReportPriorityLow, ReportPriorityMedium, ReportPriorityHigh, ReportPriorityCritical:
		return true
	}
	return false
}

func (r ReportPriority) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid report priority: %s", r)
	}
	return string(r), nil
}

func (r *ReportPriority) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ReportPriority: expected string, got %T", value)
	}
	*r = ReportPriority(str)
	if !r.IsValid() {
		return fmt.Errorf("invalid report priority value: %s", str)
	}
	return nil
}

// InviteStatus represents the status of a conversation invite
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusDeclined InviteStatus = "declined"
	InviteStatusRevoked  InviteStatus = "revoked"
	InviteStatusExpired  InviteStatus = "expired"
)

func (i InviteStatus) IsValid() bool {
	switch i {
	case InviteStatusPending, InviteStatusAccepted, InviteStatusDeclined, InviteStatusRevoked, InviteStatusExpired:
		return true
	}
	return false
}

func (i InviteStatus) Value() (driver.Value, error) {
	if !i.IsValid() {
		return nil, fmt.Errorf("invalid invite status: %s", i)
	}
	return string(i), nil
}

func (i *InviteStatus) Scan(value interface{}) error {
	if value == nil {
		*i = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan InviteStatus: expected string, got %T", value)
	}
	*i = InviteStatus(str)
	if !i.IsValid() {
		return fmt.Errorf("invalid invite status value: %s", str)
	}
	return nil
}

// PermissionLevel represents permission levels for conversation actions
type PermissionLevel string

const (
	PermissionLevelAll    PermissionLevel = "all"
	PermissionLevelAdmins PermissionLevel = "admins"
	PermissionLevelOwner  PermissionLevel = "owner"
)

func (p PermissionLevel) IsValid() bool {
	switch p {
	case PermissionLevelAll, PermissionLevelAdmins, PermissionLevelOwner:
		return true
	}
	return false
}

func (p PermissionLevel) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid permission level: %s", p)
	}
	return string(p), nil
}

func (p *PermissionLevel) Scan(value interface{}) error {
	if value == nil {
		*p = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan PermissionLevel: expected string, got %T", value)
	}
	*p = PermissionLevel(str)
	if !p.IsValid() {
		return fmt.Errorf("invalid permission level value: %s", str)
	}
	return nil
}
