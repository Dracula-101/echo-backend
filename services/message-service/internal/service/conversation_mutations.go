package service

import (
	"context"
	"time"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"

	"github.com/google/uuid"
)

// UpdateConversation updates a conversation's details
func (s *conversationService) UpdateConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, input domain.UpdateConversationInput) (*domain.Conversation, *msgError.MsgError) {
	if !input.HasChanges() {
		return nil, &msgError.MsgError{
			Message: "No fields to update",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "no fields provided"),
		}
	}

	// Verify user has edit permission
	participants, partErr := s.conversationRepo.GetConversationParticipants(ctx, conversationID)
	if partErr != nil {
		return nil, &msgError.MsgError{
			Message: "Failed to verify permissions",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   partErr,
		}
	}

	hasPermission := false
	for _, p := range participants {
		if p.UserID == userID && (p.CanEditInfo || p.HasPermissionToManage()) {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return nil, &msgError.MsgError{
			Message: "You don't have permission to update this conversation",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized"),
		}
	}

	// Check if direct conversation (cannot change title)
	convType, _ := s.conversationRepo.GetConversationType(ctx, conversationID)
	if convType == string(domain.ConversationTypeDirect) && input.Title != nil {
		return nil, &msgError.MsgError{
			Message: "Cannot change title of a direct conversation",
			Code:    msgError.CodeDirectConversationLimit,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "direct conversation title change not allowed"),
		}
	}

	if err := s.conversationRepo.UpdateConversation(ctx, conversationID, input); err != nil {
		s.logger.Error("Failed to update conversation",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to update conversation",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   err,
		}
	}

	// Publish event
	s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
		EventType:      domain.ConversationEventTypeUpdated,
		ConversationID: conversationID,
		UserID:         userID,
		Timestamp:      time.Now(),
		Payload: map[string]interface{}{
			"title":  utils.DerefString(input.Title),
			"avatar": utils.DerefString(input.AvatarURL),
		},
	})

	// Return updated conversation
	conv, fetchErr := s.conversationRepo.GetConversationByID(ctx, conversationID, &userID)
	if fetchErr != nil {
		return nil, &msgError.MsgError{
			Message: "Conversation updated but failed to fetch result",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   fetchErr,
		}
	}
	return conv, nil
}

// DeleteConversation soft-deletes a conversation (owner only)
func (s *conversationService) DeleteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) *msgError.MsgError {
	// Verify user is owner
	participants, partErr := s.conversationRepo.GetConversationParticipants(ctx, conversationID)
	if partErr != nil {
		return &msgError.MsgError{
			Message: "Failed to verify permissions",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   partErr,
		}
	}

	isOwner := false
	for _, p := range participants {
		if p.UserID == userID && p.Role == domain.ParticipantRoleOwner {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return &msgError.MsgError{
			Message: "Only the conversation owner can delete it",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not the owner"),
		}
	}

	if err := s.conversationRepo.SoftDeleteConversation(ctx, conversationID); err != nil {
		s.logger.Error("Failed to delete conversation",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to delete conversation",
			Code:    msgError.CodeConversationDeleteFailed,
			Error:   err,
		}
	}

	s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
		EventType:      domain.ConversationEventTypeDeleted,
		ConversationID: conversationID,
		UserID:         userID,
		Timestamp:      time.Now(),
	})

	return nil
}

// AddParticipants adds participants to a conversation
func (s *conversationService) AddParticipants(ctx context.Context, input domain.AddParticipantsInput) *msgError.MsgError {
	// Check conversation type - cannot add to direct conversations
	convType, typeErr := s.conversationRepo.GetConversationType(ctx, input.ConversationID)
	if typeErr != nil {
		return &msgError.MsgError{
			Message: "Failed to get conversation type",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   typeErr,
		}
	}
	if convType == string(domain.ConversationTypeDirect) {
		return &msgError.MsgError{
			Message: "Cannot add participants to a direct conversation",
			Code:    msgError.CodeDirectConversationLimit,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "direct conversation"),
		}
	}

	// Verify caller has add_members permission
	participants, partErr := s.conversationRepo.GetConversationParticipants(ctx, input.ConversationID)
	if partErr != nil {
		return &msgError.MsgError{
			Message: "Failed to verify permissions",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   partErr,
		}
	}

	hasPermission := false
	for _, p := range participants {
		if p.UserID == input.UserID && (p.CanAddMembers || p.HasPermissionToManage()) {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return &msgError.MsgError{
			Message: "You don't have permission to add participants",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized to add members"),
		}
	}

	if err := s.conversationRepo.AddParticipants(ctx, input.ConversationID, input.ParticipantIDs, input.UserID); err != nil {
		s.logger.Error("Failed to add participants",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", input.ConversationID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to add participants",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   err,
		}
	}

	s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
		EventType:      domain.ConversationEventTypeParticipantAdded,
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
		Timestamp:      time.Now(),
		Payload:        map[string]interface{}{"added_user_ids": input.ParticipantIDs},
	})

	return nil
}

// RemoveParticipant removes a participant from a conversation
func (s *conversationService) RemoveParticipant(ctx context.Context, input domain.RemoveParticipantInput) *msgError.MsgError {
	// Check conversation type
	convType, typeErr := s.conversationRepo.GetConversationType(ctx, input.ConversationID)
	if typeErr != nil {
		return &msgError.MsgError{
			Message: "Failed to get conversation type",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   typeErr,
		}
	}
	if convType == string(domain.ConversationTypeDirect) {
		return &msgError.MsgError{
			Message: "Cannot remove participants from a direct conversation",
			Code:    msgError.CodeDirectConversationLimit,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "direct conversation"),
		}
	}

	// Check if trying to remove the owner
	participants, partErr := s.conversationRepo.GetConversationParticipants(ctx, input.ConversationID)
	if partErr != nil {
		return &msgError.MsgError{
			Message: "Failed to verify permissions",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   partErr,
		}
	}

	for _, p := range participants {
		if p.UserID == input.ParticipantID && p.Role == domain.ParticipantRoleOwner {
			return &msgError.MsgError{
				Message: "Cannot remove the conversation owner",
				Code:    msgError.CodeCannotRemoveOwner,
				Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "cannot remove owner"),
			}
		}
	}

	// If removing self, allow it (leave). Otherwise check permissions.
	if input.UserID != input.ParticipantID {
		hasPermission := false
		for _, p := range participants {
			if p.UserID == input.UserID && p.HasPermissionToManage() {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return &msgError.MsgError{
				Message: "You don't have permission to remove participants",
				Code:    msgError.CodePermissionDenied,
				Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized"),
			}
		}
	}

	if err := s.conversationRepo.RemoveParticipant(ctx, input.ConversationID, input.ParticipantID, input.UserID); err != nil {
		s.logger.Error("Failed to remove participant",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", input.ConversationID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to remove participant",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   err,
		}
	}

	s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
		EventType:      domain.ConversationEventTypeParticipantLeft,
		ConversationID: input.ConversationID,
		UserID:         input.UserID,
		Timestamp:      time.Now(),
		Payload:        map[string]interface{}{"removed_user_id": input.ParticipantID},
	})

	return nil
}

// UpdateParticipantRole updates a participant's role
func (s *conversationService) UpdateParticipantRole(ctx context.Context, input domain.UpdateParticipantRoleInput) *msgError.MsgError {
	// Verify caller is owner or admin
	participants, partErr := s.conversationRepo.GetConversationParticipants(ctx, input.ConversationID)
	if partErr != nil {
		return &msgError.MsgError{
			Message: "Failed to verify permissions",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   partErr,
		}
	}

	callerIsOwner := false
	for _, p := range participants {
		if p.UserID == input.CallerUserID {
			if p.Role == domain.ParticipantRoleOwner {
				callerIsOwner = true
			} else if !p.HasPermissionToManage() {
				return &msgError.MsgError{
					Message: "You don't have permission to change roles",
					Code:    msgError.CodePermissionDenied,
					Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "not authorized"),
				}
			}
			break
		}
	}

	// Only owner can promote to admin
	if input.NewRole == "admin" && !callerIsOwner {
		return &msgError.MsgError{
			Message: "Only the owner can promote to admin",
			Code:    msgError.CodePermissionDenied,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "owner only"),
		}
	}

	if err := s.conversationRepo.UpdateParticipantRole(ctx, input.ConversationID, input.TargetUserID, input.NewRole); err != nil {
		return &msgError.MsgError{
			Message: "Failed to update participant role",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   err,
		}
	}

	s.eventPublisher.PublishConversationEvent(ctx, &domain.ConversationEvent{
		EventType:      domain.ConversationEventTypeUpdated,
		ConversationID: input.ConversationID,
		UserID:         input.CallerUserID,
		Timestamp:      time.Now(),
		Payload:        map[string]interface{}{"target_user_id": input.TargetUserID, "new_role": input.NewRole},
	})

	return nil
}

// MuteConversation mutes a conversation for a user
func (s *conversationService) MuteConversation(ctx context.Context, input domain.MuteConversationInput) *msgError.MsgError {
	if err := s.conversationRepo.MuteConversation(ctx, input.ConversationID, input.UserID, input.MuteUntil); err != nil {
		return &msgError.MsgError{
			Message: "Failed to mute conversation",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   err,
		}
	}
	return nil
}

// UnmuteConversation unmutes a conversation for a user
func (s *conversationService) UnmuteConversation(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) *msgError.MsgError {
	if err := s.conversationRepo.UnmuteConversation(ctx, conversationID, userID); err != nil {
		return &msgError.MsgError{
			Message: "Failed to unmute conversation",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   err,
		}
	}
	return nil
}

// GetUnreadCount returns unread count for a user in a conversation
func (s *conversationService) GetUnreadCount(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (int, *msgError.MsgError) {
	count, err := s.conversationRepo.GetUnreadCount(ctx, conversationID, userID)
	if err != nil {
		return 0, &msgError.MsgError{
			Message: "Failed to get unread count",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   err,
		}
	}
	return count, nil
}
