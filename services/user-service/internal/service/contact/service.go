package contact

import (
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
	"user-service/internal/errors"
	"user-service/internal/repo/contacts"
)

func (s *contactService) CheckContactById(ctx context.Context, userId string, contactId string) (*domain.Contact, pkgErrors.AppError) {
	s.log.Info("Checking contact by ID",
		logger.String("contact_id", contactId),
	)

	contact, err := s.contactRepo.GetContact(ctx, userId, contactId)
	if err != nil {
		s.log.Error("Failed to check contact by ID",
			logger.String("contact_id", contactId),
			logger.Error(err),
		)
		return nil, err
	}

	return contact, nil
}

func (s *contactService) CheckContactExists(ctx context.Context, userId string, targetUserId string) (*domain.Contact, pkgErrors.AppError) {
	s.log.Info("Checking if contact exists",
		logger.String("user_id", userId),
		logger.String("target_user_id", targetUserId),
	)

	contact, err := s.contactRepo.ContactExists(ctx, userId, targetUserId)
	if err != nil {
		s.log.Error("Failed to check if contact exists",
			logger.String("user_id", userId),
			logger.String("target_user_id", targetUserId),
			logger.Error(err),
		)
		return nil, err
	}

	return contact, nil
}

func (s *contactService) AddContact(ctx context.Context, requesterUserId string, identifier string, identifierType dto.IdentifierType, message string, source domain.ContactSource) pkgErrors.AppError {
	s.log.Info("Adding contact",
		logger.String("requester_user_id", requesterUserId),
	)

	var targetUserId string
	switch identifierType {
	case dto.IdentifierTypeUserID:
		targetUserId = identifier
	case dto.IdentifierTypePhone:
		profile, err := s.userRepo.GetProfileByPhone(ctx, identifier)
		if err != nil {
			s.log.Error("Failed to find user by phone number",
				logger.String("phone_number", identifier),
				logger.Error(err),
			)
			return pkgErrors.New(
				errors.ErrCodeUserNotFound,
				"User with the provided phone number not found",
			)
		}
		targetUserId = profile.UserID
	case dto.IdentifierTypeUsername:
		profile, err := s.userRepo.GetProfileByUsername(ctx, identifier)
		if err != nil {
			s.log.Error("Failed to find user by username",
				logger.String("username", identifier),
				logger.Error(err),
			)
			return pkgErrors.New(
				errors.ErrCodeUserNotFound,
				"User with the provided username not found",
			)
		}
		targetUserId = profile.UserID
	default:
		return pkgErrors.New(
			errors.ErrCodeForbidden,
			"Invalid identifier type",
		)
	}

	// Check if requester has blocked the target user
	blocked, err := s.blockedRepo.IsBlocked(ctx, requesterUserId, targetUserId)
	if err != nil {
		s.log.Error("Failed to check blocked status",
			logger.String("requester_user_id", requesterUserId),
			logger.String("target_user_id", targetUserId),
			logger.Error(err),
		)
		return err
	}
	if blocked {
		s.log.Info("Cannot add contact - target user is blocked",
			logger.String("requester_user_id", requesterUserId),
			logger.String("target_user_id", targetUserId),
		)
		return pkgErrors.New(
			errors.ErrCodeUserBlocked,
			"Cannot add contact - target user is blocked",
		)
	}

	// Check if target user has blocked the requester
	blocked, err = s.blockedRepo.IsBlocked(ctx, targetUserId, requesterUserId)
	if err != nil {
		s.log.Error("Failed to check if requester is blocked by target user",
			logger.String("requester_user_id", requesterUserId),
			logger.String("target_user_id", targetUserId),
			logger.Error(err),
		)
		return err
	}
	if blocked {
		s.log.Info("Cannot add contact - requester is blocked by target user",
			logger.String("requester_user_id", requesterUserId),
			logger.String("target_user_id", targetUserId),
		)
		return pkgErrors.New(
			errors.ErrCodeUserBlocked,
			"Cannot add contact - requester is blocked by target user",
		)
	}

	// Add contact
	_, err = s.contactRepo.CreateContactRequest(ctx, requesterUserId, targetUserId, message, string(source))
	if err != nil {
		s.log.Error("Failed to add contact",
			logger.String("requester_user_id", requesterUserId),
			logger.String("target_user_id", targetUserId),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Contact added successfully",
		logger.String("requester_user_id", requesterUserId),
		logger.String("target_user_id", targetUserId),
	)

	return nil
}

func (s *contactService) UpdateContact(ctx context.Context, userId string, contactId string, update domain.UpdateContact) pkgErrors.AppError {
	s.log.Info("Updating contact information",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)

	// For now, we only allow updating the contact's message. In the future, we can extend this to allow updating other fields.
	_, err := s.contactRepo.UpdateContact(ctx, contactId, userId, contacts.UpdateContactParams{
		Nickname:      update.Nickname,
		Notes:         update.Notes,
		IsFavorite:    update.IsFavorite,
		IsPinned:      update.IsPinned,
		IsArchived:    update.IsArchived,
		IsMuted:       update.IsMuted,
		MutedUntil:    update.MutedUntil,
		ContactGroups: update.ContactGroups,
	})
	if err != nil {
		s.log.Error("Failed to update contact information",
			logger.String("user_id", userId),
			logger.String("contact_id", contactId),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Contact information updated successfully",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)

	return nil
}

func (s *contactService) RemoveContact(ctx context.Context, userId string, contactId string) pkgErrors.AppError {
	s.log.Info("Removing contact",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)
	err := s.contactRepo.DeleteContact(ctx, contactId, userId)
	if err != nil {
		s.log.Error("Failed to remove contact",
			logger.String("user_id", userId),
			logger.String("contact_id", contactId),
			logger.Error(err),
		)
		return err
	}

	cacheErr := s.userCache.RemoveContact(ctx, userId, contactId)
	if cacheErr != nil {
		s.log.Error("Failed to update contact-removal cache state",
			logger.String("user_id", userId),
			logger.String("contact_id", contactId),
			logger.Error(cacheErr),
		)
	}

	// TODO: Publish user.contact.removed to Kafka: message-service will mark any shared DMs as contact_removed

	s.log.Info("Contact removed successfully",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)
	return nil
}

func (s *contactService) AcceptContact(ctx context.Context, userId string, contactId string) pkgErrors.AppError {
	s.log.Info("Accepting contact",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)

	_, err := s.contactRepo.AcceptContactRequest(ctx, contactId, userId)
	if err != nil {
		s.log.Error("Failed to accept contact",
			logger.String("user_id", userId),
			logger.String("contact_id", contactId),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Contact accepted successfully",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)

	return nil
}

func (s *contactService) RejectContact(ctx context.Context, userId string, contactId string) pkgErrors.AppError {
	s.log.Info("Rejecting contact",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)

	err := s.contactRepo.DeclineContactRequest(ctx, contactId, userId)
	if err != nil {
		s.log.Error("Failed to reject contact",
			logger.String("user_id", userId),
			logger.String("contact_id", contactId),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Contact rejected successfully",
		logger.String("user_id", userId),
		logger.String("contact_id", contactId),
	)

	return nil
}
