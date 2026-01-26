package domain

import (
	db "shared/pkg/database/postgres/models"
	"time"

	"github.com/google/uuid"
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

func NewConversationParticipant(model db.ConversationParticipant, userName, userAvatar string) *ConversationParticipant {
	return &ConversationParticipant{
		ID:                uuid.MustParse(model.ID),
		ConversationID:    uuid.MustParse(model.ConversationID),
		UserID:            uuid.MustParse(model.UserID),
		Role:              ParticipantRole(model.Role),
		JoinedAt:          model.JoinedAt,
		LeftAt:            model.LeftAt,
		LastReadAt:        model.LastReadAt,
		UnreadCount:       int(model.UnreadCount),
		IsMuted:           model.IsMuted,
		IsPinned:          model.IsPinned,
		IsArchived:        model.IsArchived,
		CanSendMessages:   model.CanSendMessages,
		CanAddMembers:     model.CanAddMembers,
		CanEditInfo:       model.CanEditInfo,
		CanDeleteMessages: model.CanDeleteMessages,
		UserName:          userName,
		UserAvatar:        userAvatar,
	}
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
