package domain

import (
	"time"

	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/utils"
)

type ContactStatus string
type RelationshipType string
type ContactSource string

const (
	ContactStatusActive   ContactStatus = "active"
	ContactStatusPending  ContactStatus = "pending"
	ContactStatusBlocked  ContactStatus = "blocked"
	ContactStatusRejected ContactStatus = "rejected"
	ContactStatusDeleted  ContactStatus = "deleted"

	RelationshipTypeFriend    RelationshipType = "friend"
	RelationshipTypeBlocked   RelationshipType = "blocked"
	RelationshipTypePending   RelationshipType = "pending"
	RelationshipTypeFollower  RelationshipType = "follower"
	RelationshipTypeFollowing RelationshipType = "following"

	ContactSourceSearch      ContactSource = "search"
	ContactSourceQRCode      ContactSource = "qr_code"
	ContactSourcePhoneImport ContactSource = "phone_import"
	ContactSourceSuggested   ContactSource = "suggested"
	ContactSourceAccept      ContactSource = "contact_accept"
)

type Contact struct {
	ID               string
	UserID           string
	ContactUserID    string
	Status           ContactStatus
	RelationshipType RelationshipType
	Nickname         *string
	Notes            *string
	IsFavorite       bool
	IsPinned         bool
	IsArchived       bool
	IsMuted          bool
	MutedUntil       *time.Time
	ContactSource    *ContactSource
	ContactGroups    []string
	AcceptedAt       *time.Time
	BlockedAt        *time.Time
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func NewContact(model dbModels.Contact) *Contact {
	return &Contact{
		ID:               model.ID,
		UserID:           model.UserID,
		ContactUserID:    model.ContactUserID,
		Status:           ContactStatus(model.Status),
		RelationshipType: RelationshipType(model.RelationshipType),
		Nickname:         model.Nickname,
		Notes:            model.Notes,
		IsFavorite:       model.IsFavorite,
		IsPinned:         model.IsPinned,
		IsArchived:       model.IsArchived,
		IsMuted:          model.IsMuted,
		MutedUntil:       model.MutedUntil,
		ContactSource: func() *ContactSource {
			if model.ContactSource == nil {
				return nil
			}
			return utils.Ptr(ContactSource(*model.ContactSource))
		}(),
		ContactGroups: []string(model.ContactGroups),
		AcceptedAt:    model.AcceptedAt,
		BlockedAt:     model.BlockedAt,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

type UpdateContact struct {
	Nickname      *string    `json:"nickname,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	IsFavorite    *bool      `json:"is_favorite,omitempty"`
	IsPinned      *bool      `json:"is_pinned,omitempty"`
	IsArchived    *bool      `json:"is_archived,omitempty"`
	IsMuted       *bool      `json:"is_muted,omitempty"`
	MutedUntil    *time.Time `json:"muted_until,omitempty"`
	ContactGroups *[]string  `json:"contact_groups,omitempty"`
}
