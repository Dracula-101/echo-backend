package service

import (
	"context"
	"time"

	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// DraftResponse represents the response payload for a draft.
type DraftResponse struct {
	ID               string    `json:"id"`
	ConversationID   string    `json:"conversation_id"`
	Content          string    `json:"content"`
	ReplyToMessageID *string   `json:"reply_to_message_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DraftServiceInterface defines the interface for draft business logic.
type DraftServiceInterface interface {
	UpsertDraft(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, content string, replyToMessageID *uuid.UUID) *msgError.MsgError
	GetDraft(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (*DraftResponse, *msgError.MsgError)
	DeleteDraft(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) *msgError.MsgError
}

type draftService struct {
	draftRepo repo.DraftRepository
	logger    logger.Logger
}

// NewDraftService creates a new instance of DraftServiceInterface.
func NewDraftService(draftRepo repo.DraftRepository, log logger.Logger) DraftServiceInterface {
	return &draftService{
		draftRepo: draftRepo,
		logger:    log,
	}
}

// UpsertDraft creates or updates a draft for the given user and conversation.
func (s *draftService) UpsertDraft(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, content string, replyToMessageID *uuid.UUID) *msgError.MsgError {
	err := s.draftRepo.UpsertDraft(ctx, userID, conversationID, content, replyToMessageID)
	if err != nil {
		s.logger.Error("Failed to upsert draft",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to upsert draft",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to upsert draft"),
		}
	}

	return nil
}

// GetDraft retrieves a draft for the given user and conversation. Returns nil DraftResponse if not found.
func (s *draftService) GetDraft(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (*DraftResponse, *msgError.MsgError) {
	draft, err := s.draftRepo.GetDraft(ctx, userID, conversationID)
	if err != nil {
		s.logger.Error("Failed to get draft",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to get draft",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get draft"),
		}
	}

	if draft == nil {
		return nil, nil
	}

	response := &DraftResponse{
		ID:             draft.ID.String(),
		ConversationID: draft.ConversationID.String(),
		Content:        draft.Content,
		CreatedAt:      draft.CreatedAt,
		UpdatedAt:      draft.UpdatedAt,
	}

	if draft.ReplyToMessageID != nil {
		replyID := draft.ReplyToMessageID.String()
		response.ReplyToMessageID = &replyID
	}

	return response, nil
}

// DeleteDraft removes a draft for the given user and conversation.
func (s *draftService) DeleteDraft(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) *msgError.MsgError {
	err := s.draftRepo.DeleteDraft(ctx, userID, conversationID)
	if err != nil {
		s.logger.Error("Failed to delete draft",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Message: "Failed to delete draft",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to delete draft"),
		}
	}

	return nil
}
