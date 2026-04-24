package dto

import (
	"time"
	"user-service/internal/domain"
)

type ContactListItem struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	ContactUserID     string     `json:"contact_user_id"`
	Status            string     `json:"status"`
	RelationshipType  string     `json:"relationship_type"`
	Nickname          *string    `json:"nickname,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	IsFavorite        bool       `json:"is_favorite"`
	IsPinned          bool       `json:"is_pinned"`
	IsArchived        bool       `json:"is_archived"`
	IsMuted           bool       `json:"is_muted"`
	ContactGroups     []string   `json:"contact_groups,omitempty"`
	DisplayName       *string    `json:"display_name,omitempty"`
	Username          *string    `json:"username,omitempty"`
	AvatarURL         *string    `json:"avatar_url,omitempty"`
	OnlineStatus      *string    `json:"online_status,omitempty"`
	LastInteractionAt *time.Time `json:"last_interaction_at,omitempty"`
	CreatedAt         *time.Time `json:"created_at,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type ListContactsResponse struct {
	Contacts            []ContactListItem `json:"contacts"`
	Total               int               `json:"total"`
	Page                int               `json:"page"`
	Limit               int               `json:"limit"`
	UnreadRequestsCount *int              `json:"unread_requests_count,omitempty"`
}

func NewListContactsResponse(contacts []*domain.Contact, total, page, limit int, unreadRequestsCount *int) *ListContactsResponse {
	items := make([]ContactListItem, len(contacts))
	for i, c := range contacts {
		items[i] = ContactListItem{
			ID:                c.ID,
			UserID:            c.UserID,
			ContactUserID:     c.ContactUserID,
			Status:            string(c.Status),
			RelationshipType:  string(c.RelationshipType),
			Nickname:          c.Nickname,
			Notes:             c.Notes,
			IsFavorite:        c.IsFavorite,
			IsPinned:          c.IsPinned,
			IsArchived:        c.IsArchived,
			IsMuted:           c.IsMuted,
			ContactGroups:     c.ContactGroups,
			DisplayName:       c.DisplayName,
			Username:          c.Username,
			AvatarURL:         c.AvatarURL,
			OnlineStatus:      c.OnlineStatus,
			LastInteractionAt: c.LastInteractionAt,
			CreatedAt:         c.CreatedAt,
			UpdatedAt:         c.UpdatedAt,
		}
	}

	return &ListContactsResponse{
		Contacts:            items,
		Total:               total,
		Page:                page,
		Limit:               limit,
		UnreadRequestsCount: unreadRequestsCount,
	}
}
