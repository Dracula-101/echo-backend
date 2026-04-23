package contacts

import (
	"context"
	"time"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

type GetContactsParams struct {
	Status       string
	Relationship string
	GroupID      *string
	Sort         string
	Search       *string
	Limit        int
	Offset       int
}

type UpdateContactParams struct {
	Nickname      *string
	Notes         *string
	IsFavorite    *bool
	IsPinned      *bool
	IsArchived    *bool
	IsMuted       *bool
	MutedUntil    *time.Time
	ContactGroups *[]string
}

type ContactRepository interface {
	GetContactIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError)
	GetContacts(ctx context.Context, userID string, params GetContactsParams) ([]*domain.Contact, int, pkgErrors.AppError)
	GetContact(ctx context.Context, userID, contactID string) (*domain.Contact, pkgErrors.AppError)
	RemoveContact(ctx context.Context, contactID, userID string) pkgErrors.AppError
	GetPendingContactRequests(ctx context.Context, userID string) ([]*domain.Contact, pkgErrors.AppError)
	ContactExists(ctx context.Context, userID, targetID string) (*domain.Contact, pkgErrors.AppError)
	CreateContactRequest(ctx context.Context, userID, targetID, message, source string) (*domain.Contact, pkgErrors.AppError)
	AcceptContactRequest(ctx context.Context, contactID, userID string) (*domain.Contact, pkgErrors.AppError)
	DeclineContactRequest(ctx context.Context, contactID, userID string) pkgErrors.AppError
	UpdateContact(ctx context.Context, contactID, userID string, params UpdateContactParams) (*domain.Contact, pkgErrors.AppError)
	DeleteContact(ctx context.Context, contactID, userID string) pkgErrors.AppError
}

var _ ContactRepository = (*contactRepository)(nil)
