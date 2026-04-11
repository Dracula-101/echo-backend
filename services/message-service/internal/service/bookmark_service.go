package service

import (
	"context"
	"time"

	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// BookmarkResponse represents a bookmark returned to callers.
type BookmarkResponse struct {
	ID           string
	MessageID    string
	Notes        *string
	BookmarkedAt time.Time
}

// BookmarkServiceInterface defines the interface for bookmark operations.
type BookmarkServiceInterface interface {
	CreateBookmark(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, notes *string) *msgError.MessageError
	DeleteBookmark(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) *msgError.MessageError
	GetBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]BookmarkResponse, int, *msgError.MessageError)
}

type bookmarkService struct {
	bookmarkRepo repo.BookmarkRepository
	logger       logger.Logger
}

// NewBookmarkService creates a new instance of BookmarkServiceInterface.
func NewBookmarkService(bookmarkRepo repo.BookmarkRepository, log logger.Logger) BookmarkServiceInterface {
	return &bookmarkService{
		bookmarkRepo: bookmarkRepo,
		logger:       log,
	}
}

// CreateBookmark adds a bookmark for the given message and user.
func (s *bookmarkService) CreateBookmark(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, notes *string) *msgError.MessageError {
	if err := s.bookmarkRepo.CreateBookmark(ctx, userID, messageID, notes); err != nil {
		s.logger.Error("Failed to create bookmark",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return &msgError.MessageError{
			Code:    msgError.CodeConversationUpdateFailed,
			Message: "Failed to create bookmark",
		}
	}

	return nil
}

// DeleteBookmark removes a bookmark for the given message and user.
func (s *bookmarkService) DeleteBookmark(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) *msgError.MessageError {
	if err := s.bookmarkRepo.DeleteBookmark(ctx, userID, messageID); err != nil {
		s.logger.Error("Failed to delete bookmark",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return &msgError.MessageError{
			Code:    msgError.CodeConversationUpdateFailed,
			Message: "Failed to delete bookmark",
		}
	}

	return nil
}

// GetBookmarks retrieves bookmarks for a user with pagination.
func (s *bookmarkService) GetBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]BookmarkResponse, int, *msgError.MessageError) {
	bookmarks, total, err := s.bookmarkRepo.GetBookmarks(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to fetch bookmarks",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, 0, &msgError.MessageError{
			Code:    msgError.CodeConversationFetchFailed,
			Message: "Failed to fetch bookmarks",
		}
	}

	responses := make([]BookmarkResponse, len(bookmarks))
	for i, b := range bookmarks {
		responses[i] = BookmarkResponse{
			ID:           b.ID.String(),
			MessageID:    b.MessageID.String(),
			Notes:        b.Notes,
			BookmarkedAt: b.BookmarkedAt,
		}
	}

	return responses, total, nil
}
