package repo

import (
	"context"
	"shared/pkg/cache"
	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"
)

type ConversationRepository interface {
	ValidateConversationParticipant(conversationID string, userID string) (bool, pkgErrors.AppError)
}

type conversationRepository struct {
	db    database.Database
	cache cache.Cache
}

func NewConversationRepository(db database.Database, cache cache.Cache) ConversationRepository {
	return &conversationRepository{db: db, cache: cache}
}

func (r *conversationRepository) ValidateConversationParticipant(conversationID string, userID string) (bool, pkgErrors.AppError) {
	query := `
		SELECT COUNT(1)
		FROM conversation_participants
		WHERE conversation_id = $1 AND user_id = $2
	`
	var count int
	err := r.db.QueryRow(context.Background(), query, conversationID, userID).Scan(&count)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to validate conversation participant")
	}
	return count > 0, nil
}
