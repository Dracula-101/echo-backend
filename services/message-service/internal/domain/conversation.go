package domain

import (
	"time"

	"github.com/google/uuid"
)

// Conversation represents a messaging conversation
type Conversation struct {
	ID               uuid.UUID
	ConversationType ConversationType
	Title            string
	Description      string
	AvatarURL        *string
	CreatorUserID    uuid.UUID
	IsEncrypted      bool
	IsPublic         bool
	IsArchived       bool
	MemberCount      int
	MessageCount     int
	LastMessageID    *uuid.UUID
	LastMessageAt    *time.Time
	LastActivityAt   *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time

	// Enriched fields (from joins)
	Participants []ConversationParticipant
	UnreadCount  int
	LastMessage  *Message
}

// ConversationType represents the type of conversation
type ConversationType string

const (
	ConversationTypeDirect    ConversationType = "direct"
	ConversationTypeGroup     ConversationType = "group"
	ConversationTypeChannel   ConversationType = "channel"
	ConversationTypeBroadcast ConversationType = "broadcast"
)

// ConversationParticipant represents a user in a conversation
type ConversationParticipant struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Role           ParticipantRole
	JoinedAt       time.Time
	LeftAt         *time.Time
	LastReadAt     *time.Time
	UnreadCount    int
	IsMuted        bool
	IsPinned       bool
	IsArchived     bool

	// Permissions
	CanSendMessages   bool
	CanAddMembers     bool
	CanEditInfo       bool
	CanDeleteMessages bool

	// Enriched fields
	UserName   string
	UserAvatar string
}

// ParticipantRole defines the role of a participant
type ParticipantRole string

const (
	ParticipantRoleOwner  ParticipantRole = "owner"
	ParticipantRoleAdmin  ParticipantRole = "admin"
	ParticipantRoleMember ParticipantRole = "member"
	ParticipantRoleViewer ParticipantRole = "viewer"
)

// IsActive checks if participant is still in conversation
func (p *ConversationParticipant) IsActive() bool {
	return p.LeftAt == nil
}

// HasPermissionToSend checks if participant can send messages
func (p *ConversationParticipant) HasPermissionToSend() bool {
	return p.IsActive() && p.CanSendMessages
}

// HasPermissionToManage checks if participant can manage conversation
func (p *ConversationParticipant) HasPermissionToManage() bool {
	return p.IsActive() && (p.Role == ParticipantRoleOwner || p.Role == ParticipantRoleAdmin)
}

// Conversation business logic methods

// IsDirect checks if conversation is a direct message
func (c *Conversation) IsDirect() bool {
	return c.ConversationType == ConversationTypeDirect
}

// IsGroup checks if conversation is a group chat
func (c *Conversation) IsGroup() bool {
	return c.ConversationType == ConversationTypeGroup
}

// IsDeleted checks if conversation is soft deleted
func (c *Conversation) IsDeleted() bool {
	return c.DeletedAt != nil
}

// CanUserSendMessage checks if a user can send messages
func (c *Conversation) CanUserSendMessage(userID uuid.UUID) error {
	if c.IsDeleted() {
		return ErrConversationDeleted
	}

	if c.IsArchived {
		return ErrConversationArchived
	}

	participant := c.GetParticipant(userID)
	if participant == nil {
		return ErrNotParticipant
	}

	if !participant.HasPermissionToSend() {
		return ErrNoSendPermission
	}

	return nil
}

// GetParticipant finds a participant by user ID
func (c *Conversation) GetParticipant(userID uuid.UUID) *ConversationParticipant {
	for i := range c.Participants {
		if c.Participants[i].UserID == userID && c.Participants[i].IsActive() {
			return &c.Participants[i]
		}
	}
	return nil
}

// GetActiveParticipants returns all active participants
func (c *Conversation) GetActiveParticipants() []ConversationParticipant {
	active := make([]ConversationParticipant, 0, len(c.Participants))
	for _, p := range c.Participants {
		if p.IsActive() {
			active = append(active, p)
		}
	}
	return active
}

// GetParticipantUserIDs returns user IDs of active participants
func (c *Conversation) GetParticipantUserIDs() []uuid.UUID {
	participants := c.GetActiveParticipants()
	userIDs := make([]uuid.UUID, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserID
	}
	return userIDs
}

// HasActivity checks if conversation had recent activity
func (c *Conversation) HasActivity(duration time.Duration) bool {
	if c.LastActivityAt == nil {
		return false
	}
	cutoff := time.Now().Add(-duration)
	return c.LastActivityAt.After(cutoff)
}

// ShouldNotifyUser checks if user should be notified
func (c *Conversation) ShouldNotifyUser(userID uuid.UUID) bool {
	participant := c.GetParticipant(userID)
	if participant == nil {
		return false
	}
	return !participant.IsMuted
}
