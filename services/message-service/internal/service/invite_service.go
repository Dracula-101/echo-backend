package service

import (
	"context"
	"crypto/rand"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	"encoding/hex"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"time"

	"github.com/google/uuid"
)

// InviteResponse is the service-layer representation of a conversation invite.
type InviteResponse struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	InviteCode     string     `json:"invite_code"`
	Status         string     `json:"status"`
	MaxUses        int        `json:"max_uses"`
	UseCount       int        `json:"use_count"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// InviteServiceInterface defines methods for managing conversation invites.
type InviteServiceInterface interface {
	CreateInvite(ctx context.Context, conversationID, userID uuid.UUID, maxUses int, expiresAt *time.Time) (*InviteResponse, *msgError.MsgError)
	AcceptInvite(ctx context.Context, inviteCode string, userID uuid.UUID) *msgError.MsgError
	GetConversationInvites(ctx context.Context, conversationID uuid.UUID) ([]InviteResponse, *msgError.MsgError)
	RevokeInvite(ctx context.Context, inviteID uuid.UUID, userID uuid.UUID) *msgError.MsgError
}

type inviteService struct {
	inviteRepo       repo.InviteRepository
	conversationRepo repo.ConversationRepository
	logger           logger.Logger
}

func NewInviteService(inviteRepo repo.InviteRepository, conversationRepo repo.ConversationRepository, log logger.Logger) InviteServiceInterface {
	return &inviteService{
		inviteRepo:       inviteRepo,
		conversationRepo: conversationRepo,
		logger:           log,
	}
}

func generateInviteCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *inviteService) CreateInvite(ctx context.Context, conversationID, userID uuid.UUID, maxUses int, expiresAt *time.Time) (*InviteResponse, *msgError.MsgError) {
	inviteCode := generateInviteCode()

	invite, err := s.inviteRepo.CreateInvite(ctx, conversationID, userID, inviteCode, maxUses, expiresAt)
	if err != nil {
		s.logger.Error("Failed to create invite",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to create invite",
			Code:    msgError.CodeConversationCreationFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to create invite"),
		}
	}

	return &InviteResponse{
		ID:             invite.ID.String(),
		ConversationID: invite.ConversationID.String(),
		InviteCode:     invite.InviteCode,
		Status:         invite.Status,
		MaxUses:        invite.MaxUses,
		UseCount:       invite.UseCount,
		ExpiresAt:      invite.ExpiresAt,
		CreatedAt:      invite.CreatedAt,
	}, nil
}

func (s *inviteService) AcceptInvite(ctx context.Context, inviteCode string, userID uuid.UUID) *msgError.MsgError {
	invite, err := s.inviteRepo.GetInviteByCode(ctx, inviteCode)
	if err != nil {
		s.logger.Error("Failed to get invite by code",
			logger.String("service", msgError.ServiceName),
			logger.String("invite_code", inviteCode),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Invite not found",
			Code:    msgError.CodeConversationNotFound,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeNotFound, "Invite not found"),
		}
	}

	if invite == nil {
		return &msgError.MsgError{
			Message: "Invite not found",
			Code:    msgError.CodeConversationNotFound,
			Error:   pkgErrors.New(pkgErrors.CodeNotFound, "Invite not found"),
		}
	}

	if invite.Status == "revoked" {
		return &msgError.MsgError{
			Message: "Invite has been revoked",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "Invite has been revoked"),
		}
	}

	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return &msgError.MsgError{
			Message: "Invite has expired",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "Invite has expired"),
		}
	}

	if invite.UseCount >= invite.MaxUses {
		return &msgError.MsgError{
			Message: "Invite has reached maximum uses",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   pkgErrors.New(pkgErrors.CodeBadRequest, "Invite has reached maximum uses"),
		}
	}

	if err := s.inviteRepo.IncrementUseCount(ctx, invite.ID); err != nil {
		s.logger.Error("Failed to increment invite use count",
			logger.String("service", msgError.ServiceName),
			logger.String("invite_id", invite.ID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to accept invite",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to increment invite use count"),
		}
	}

	return nil
}

func (s *inviteService) GetConversationInvites(ctx context.Context, conversationID uuid.UUID) ([]InviteResponse, *msgError.MsgError) {
	invites, err := s.inviteRepo.GetConversationInvites(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to get conversation invites",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to get conversation invites",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get conversation invites"),
		}
	}

	responses := make([]InviteResponse, len(invites))
	for i, invite := range invites {
		responses[i] = InviteResponse{
			ID:             invite.ID.String(),
			ConversationID: invite.ConversationID.String(),
			InviteCode:     invite.InviteCode,
			Status:         invite.Status,
			MaxUses:        invite.MaxUses,
			UseCount:       invite.UseCount,
			ExpiresAt:      invite.ExpiresAt,
			CreatedAt:      invite.CreatedAt,
		}
	}

	return responses, nil
}

func (s *inviteService) RevokeInvite(ctx context.Context, inviteID uuid.UUID, userID uuid.UUID) *msgError.MsgError {
	if err := s.inviteRepo.RevokeInvite(ctx, inviteID); err != nil {
		s.logger.Error("Failed to revoke invite",
			logger.String("service", msgError.ServiceName),
			logger.String("invite_id", inviteID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to revoke invite",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to revoke invite"),
		}
	}

	return nil
}
