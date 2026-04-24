package contact

import (
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"strings"
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

func (s *contactService) GetContacts(ctx context.Context, userId string, params GetContactsParams) (*GetContactsResult, pkgErrors.AppError) {
	s.log.Info("Retrieving contacts for user",
		logger.String("user_id", userId),
	)

	status := domain.ContactStatus(params.Status)
	if status == "" {
		status = domain.ContactStatusActive
	}
	if strings.EqualFold(string(status), "accepted") {
		status = domain.ContactStatusActive
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	repoParams := contacts.GetContactsParams{
		Status:       status,
		Relationship: strings.TrimSpace(params.Relationship),
		GroupID:      params.GroupID,
		Search:       params.Search,
		Limit:        limit,
		Offset:       offset,
	}

	list, total, err := s.contactRepo.GetContacts(ctx, userId, repoParams)
	if err != nil {
		s.log.Error("Failed to retrieve contacts for user",
			logger.String("user_id", userId),
			logger.Error(err),
		)
		return nil, err
	}

	isDefaultAcceptedView := status == domain.ContactStatusActive &&
		repoParams.Relationship == "" &&
		repoParams.GroupID == nil &&
		repoParams.Search == nil

	membership := map[string]bool{}
	if isDefaultAcceptedView {
		cachedIDs, cacheErr := s.userCache.GetContactIDs(ctx, userId)
		if cacheErr != nil {
			s.log.Error("Failed to get cached contact IDs",
				logger.String("user_id", userId),
				logger.Error(cacheErr),
			)
		}

		contactIDs := []string{}
		if cachedIDs != nil {
			contactIDs = *cachedIDs
		} else {
			ids, idsErr := s.contactRepo.GetContactIDs(ctx, userId)
			if idsErr != nil {
				s.log.Error("Failed to get contact IDs from repository",
					logger.String("user_id", userId),
					logger.Error(idsErr),
				)
				return nil, idsErr
			}
			contactIDs = ids
			if setErr := s.userCache.SetContactIDs(ctx, userId, ids); setErr != nil {
				s.log.Error("Failed to cache contact IDs",
					logger.String("user_id", userId),
					logger.Error(setErr),
				)
			}
		}

		for _, id := range contactIDs {
			membership[id] = true
		}
	}

	contactUserIDs := make([]string, 0, len(list))
	for _, c := range list {
		contactUserIDs = append(contactUserIDs, c.ContactUserID)
	}

	presenceByUserID, presenceErr := s.userCache.GetPresences(ctx, contactUserIDs)
	if presenceErr != nil {
		s.log.Error("Failed to batch load contact presences",
			logger.String("user_id", userId),
			logger.Error(presenceErr),
		)
	}

	for _, c := range list {
		areContacts := membership[c.ContactUserID] || status == domain.ContactStatusActive
		canSeeOnline := canViewOnlineStatus(c.OnlineStatusVisibility, areContacts)
		if !canSeeOnline {
			c.OnlineStatus = nil
			continue
		}

		if presence, ok := presenceByUserID[c.ContactUserID]; ok && presence != nil {
			statusVal := presence.Status
			c.OnlineStatus = &statusVal
		}
	}

	var unreadRequestsCount *int
	if status == domain.ContactStatusActive {
		count, countErr := s.contactRepo.GetPendingIncomingRequestsCount(ctx, userId)
		if countErr != nil {
			s.log.Error("Failed to count pending incoming contact requests",
				logger.String("user_id", userId),
				logger.Error(countErr),
			)
			return nil, countErr
		}
		unreadRequestsCount = &count
	}

	s.log.Info("Contacts retrieved successfully",
		logger.String("user_id", userId),
		logger.Int("contact_count", len(list)),
	)

	return &GetContactsResult{
		Contacts:            list,
		Total:               total,
		Page:                page,
		Limit:               limit,
		UnreadRequestsCount: unreadRequestsCount,
	}, nil
}

func canViewOnlineStatus(visibility *string, areContacts bool) bool {
	if visibility == nil {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(*visibility)) {
	case "public":
		return true
	case "friends":
		return areContacts
	case "private":
		return false
	default:
		return false
	}
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
