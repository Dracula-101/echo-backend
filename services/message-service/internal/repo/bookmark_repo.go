package repo

import (
	"context"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Bookmark represents a user's bookmarked message.
type Bookmark struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	MessageID    uuid.UUID
	Notes        *string
	BookmarkedAt time.Time
}

// BookmarkRepository defines the interface for bookmark interactions.
type BookmarkRepository interface {
	CreateBookmark(ctx context.Context, userID, messageID uuid.UUID, notes *string) error
	DeleteBookmark(ctx context.Context, userID, messageID uuid.UUID) error
	GetBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Bookmark, int, error)
}

type bookmarkRepository struct {
	db     database.Database
	logger logger.Logger
}

// NewBookmarkRepository creates a new instance of BookmarkRepository.
func NewBookmarkRepository(db database.Database, log logger.Logger) BookmarkRepository {
	return &bookmarkRepository{
		db:     db,
		logger: log,
	}
}

// CreateBookmark inserts a new bookmark. Returns an error on conflict.
func (r *bookmarkRepository) CreateBookmark(ctx context.Context, userID, messageID uuid.UUID, notes *string) error {
	query := `
		INSERT INTO messages.bookmarks (user_id, message_id, notes)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, userID, messageID, notes)
	if err != nil {
		r.logger.Error("Failed to create bookmark",
			logger.String("user_id", userID.String()),
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		return err
	}

	return nil
}

// DeleteBookmark removes a bookmark for the given user and message.
func (r *bookmarkRepository) DeleteBookmark(ctx context.Context, userID, messageID uuid.UUID) error {
	query := `
		DELETE FROM messages.bookmarks
		WHERE user_id = $1 AND message_id = $2
	`

	_, err := r.db.Exec(ctx, query, userID, messageID)
	if err != nil {
		r.logger.Error("Failed to delete bookmark",
			logger.String("user_id", userID.String()),
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		return err
	}

	return nil
}

// GetBookmarks retrieves bookmarks for a user with pagination. Returns bookmarks, total count, and error.
func (r *bookmarkRepository) GetBookmarks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Bookmark, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM messages.bookmarks
		WHERE user_id = $1
	`

	var total int
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		r.logger.Error("Failed to count bookmarks",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, message_id, notes, bookmarked_at
		FROM messages.bookmarks
		WHERE user_id = $1
		ORDER BY bookmarked_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to fetch bookmarks",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, 0, err
	}
	defer rows.Close()

	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if scanErr := rows.Scan(
			&b.ID,
			&b.UserID,
			&b.MessageID,
			&b.Notes,
			&b.BookmarkedAt,
		); scanErr != nil {
			r.logger.Error("Failed to scan bookmark row",
				logger.String("user_id", userID.String()),
				logger.Error(scanErr),
			)
			return nil, 0, scanErr
		}
		bookmarks = append(bookmarks, b)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return bookmarks, total, nil
}
