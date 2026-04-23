package contact

import (
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
	repository "user-service/internal/repo"
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/contacts"
	"user-service/internal/service/cache"
)

type ContactService interface {
	CheckContactById(ctx context.Context, userId string, contactId string) (*domain.Contact, pkgErrors.AppError)
	CheckContactExists(ctx context.Context, userId string, targetUserId string) (*domain.Contact, pkgErrors.AppError)
	AddContact(ctx context.Context, requesterUserId string, identifier string, identifierType dto.IdentifierType, message string, source domain.ContactSource) pkgErrors.AppError
	UpdateContact(ctx context.Context, userId string, contactId string, update domain.UpdateContact) pkgErrors.AppError
	RemoveContact(ctx context.Context, userId string, contactId string) pkgErrors.AppError
	AcceptContact(ctx context.Context, userId string, contactId string) pkgErrors.AppError
	RejectContact(ctx context.Context, userId string, contactId string) pkgErrors.AppError
}

type contactService struct {
	contactRepo contacts.ContactRepository
	userRepo    repository.UserRepository
	blockedRepo blocked_users.BlockedRepository
	userCache   cache.UserCache
	log         logger.Logger
}

func NewContactService(contactRepo contacts.ContactRepository, userRepo repository.UserRepository, blockedRepo blocked_users.BlockedRepository, cache cache.UserCache, log logger.Logger) ContactService {
	return &contactService{
		contactRepo: contactRepo,
		userRepo:    userRepo,
		blockedRepo: blockedRepo,
		userCache:   cache,
		log:         log,
	}
}

// Compile-time interface compliance check
var _ ContactService = (*contactService)(nil)
