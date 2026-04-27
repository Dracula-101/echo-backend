package service

import (
	"context"
	"time"

	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

type BookmarkResponse struct {
	ID             string
	MessageID      string
	CollectionName *string
	Notes          *string
	Tags           []string
	BookmarkedAt   time.Time
}

type CreateBookmarkInput struct {
	MessageID      uuid.UUID
	UserID         uuid.UUID
	CollectionName *string
	Notes          *string
	Tags           []string
}

type BookmarkServiceInterface interface {
	CreateBookmark(ctx context.Context, input CreateBookmarkInput) *msgError.MsgError
	DeleteBookmark(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) *msgError.MsgError
	GetBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]BookmarkResponse, int, *msgError.MsgError)
}

type bookmarkService struct {
	bookmarkRepo repo.BookmarkRepository
	logger       logger.Logger
}

func NewBookmarkService(bookmarkRepo repo.BookmarkRepository, log logger.Logger) BookmarkServiceInterface {
	return &bookmarkService{
		bookmarkRepo: bookmarkRepo,
		logger:       log,
	}
}

func (s *bookmarkService) CreateBookmark(ctx context.Context, in CreateBookmarkInput) *msgError.MsgError {
	if err := s.bookmarkRepo.CreateBookmark(ctx, repo.CreateBookmarkInput{
		UserID:         in.UserID,
		MessageID:      in.MessageID,
		CollectionName: in.CollectionName,
		Notes:          in.Notes,
		Tags:           in.Tags,
	}); err != nil {
		s.logger.Error("Failed to create bookmark",
			logger.String("message_id", in.MessageID.String()),
			logger.String("user_id", in.UserID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Code:    msgError.CodeConversationUpdateFailed,
			Message: "Failed to create bookmark",
		}
	}
	return nil
}

func (s *bookmarkService) DeleteBookmark(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) *msgError.MsgError {
	if err := s.bookmarkRepo.DeleteBookmark(ctx, userID, messageID); err != nil {
		s.logger.Error("Failed to delete bookmark",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Code:    msgError.CodeConversationUpdateFailed,
			Message: "Failed to delete bookmark",
		}
	}
	return nil
}

func (s *bookmarkService) GetBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]BookmarkResponse, int, *msgError.MsgError) {
	bookmarks, total, err := s.bookmarkRepo.GetBookmarks(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to fetch bookmarks",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, 0, &msgError.MsgError{
			Code:    msgError.CodeConversationFetchFailed,
			Message: "Failed to fetch bookmarks",
		}
	}

	responses := make([]BookmarkResponse, len(bookmarks))
	for i, b := range bookmarks {
		responses[i] = BookmarkResponse{
			ID:             b.ID.String(),
			MessageID:      b.MessageID.String(),
			CollectionName: b.CollectionName,
			Notes:          b.Notes,
			Tags:           b.Tags,
			BookmarkedAt:   b.BookmarkedAt,
		}
	}

	return responses, total, nil
}
