package contact

import (
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
	"user-service/internal/errors"
)

func (s *contactService) CheckContactById(ctx context.Context, userId string, contactId string) (*domain.Contact, error) {
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

func (s *contactService) CheckContactExists(ctx context.Context, userId string, targetUserId string) (*domain.Contact, error) {
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

func (s *contactService) AddContact(ctx context.Context, requesterUserId string, identifier string, identifierType dto.IdentifierType, message string, source domain.ContactSource) error {
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

func (s *contactService) AcceptContact(ctx context.Context, userId string, contactId string) error {
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

func (s *contactService) RejectContact(ctx context.Context, userId string, contactId string) error {
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
