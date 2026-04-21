package contact

import (
	"context"
	"shared/pkg/logger"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
	repository "user-service/internal/repo"
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/contacts"
)

type ContactService interface {
	CheckContactById(ctx context.Context, userId string, contactId string) (*domain.Contact, error)
	CheckContactExists(ctx context.Context, userId string, targetUserId string) (*domain.Contact, error)
	AddContact(ctx context.Context, requesterUserId string, identifier string, identifierType dto.IdentifierType, message string, source domain.ContactSource) error
	AcceptContact(ctx context.Context, userId string, contactId string) error
	RejectContact(ctx context.Context, userId string, contactId string) error
}

type contactService struct {
	contactRepo contacts.ContactRepository
	userRepo    repository.UserRepository
	blockedRepo blocked_users.BlockedRepository
	log         logger.Logger
}

func NewContactService(contactRepo contacts.ContactRepository, userRepo repository.UserRepository, blockedRepo blocked_users.BlockedRepository, log logger.Logger) ContactService {
	return &contactService{
		contactRepo: contactRepo,
		userRepo:    userRepo,
		blockedRepo: blockedRepo,
		log:         log,
	}
}

// Compile-time interface compliance check
var _ ContactService = (*contactService)(nil)
