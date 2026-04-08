package repo

import (
	"context"
	"shared/pkg/cache"
	"shared/pkg/database"
	dbModel "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// ReactionRepository defines the interface for message reactions interactions.
type ReactionRepository interface {
	AddReaction(ctx context.Context, reaction *dbModel.Reaction) pkgErrors.AppError
	RemoveReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reactionType string) pkgErrors.AppError
	GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]dbModel.Reaction, pkgErrors.AppError)
}

type reactionRepository struct {
	db     database.Database
	cache  cache.Cache
	logger logger.Logger
}

// NewReactionRepository creates a new instance of ReactionRepository.
func NewReactionRepository(db database.Database, cache cache.Cache, log logger.Logger) ReactionRepository {
	return &reactionRepository{
		db:     db,
		cache:  cache,
		logger: log,
	}
}

// AddReaction inserts a new reaction. If a reaction of the same type by the user already exists, it is ignored.
func (r *reactionRepository) AddReaction(ctx context.Context, reaction *dbModel.Reaction) pkgErrors.AppError {
	query := `
		INSERT INTO messages.reactions (
			id, message_id, user_id, reaction_type, 
			reaction_emoji, reaction_skin_tone, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (message_id, user_id, reaction_type) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query,
		reaction.ID,
		reaction.MessageID,
		reaction.UserID,
		reaction.ReactionType,
		reaction.ReactionEmoji,
		reaction.ReactionSkinTone,
		reaction.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to add reaction",
			logger.String("message_id", reaction.MessageID),
			logger.String("user_id", reaction.UserID),
			logger.String("reaction_type", reaction.ReactionType),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to add reaction")
	}

	updateMsgQuery := `
		UPDATE messages.messages
		SET reaction_count = reaction_count + 1
		WHERE id = $1
	`
	if _, updateErr := r.db.Exec(ctx, updateMsgQuery, reaction.MessageID); updateErr != nil {
		r.logger.Error("Failed to increment message reaction count",
			logger.String("message_id", reaction.MessageID),
			logger.Error(updateErr),
		)
	}

	return nil
}

// RemoveReaction removes a specific reaction by a user on a message.
func (r *reactionRepository) RemoveReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reactionType string) pkgErrors.AppError {
	query := `
		DELETE FROM messages.reactions
		WHERE message_id = $1 AND user_id = $2 AND reaction_type = $3
	`

	res, err := r.db.Exec(ctx, query, messageID, userID, reactionType)
	if err != nil {
		r.logger.Error("Failed to remove reaction",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.String("reaction_type", reactionType),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to remove reaction")
	}

	rowsAffected, affErr := res.RowsAffected()
	if affErr != nil {
		return pkgErrors.FromError(affErr, pkgErrors.CodeDatabaseError, "Failed to get rows affected")
	}
	if rowsAffected == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "reaction not found")
	}

	updateMsgQuery := `
		UPDATE messages.messages
		SET reaction_count = GREATEST(reaction_count - 1, 0)
		WHERE id = $1
	`
	if _, updateErr := r.db.Exec(ctx, updateMsgQuery, messageID); updateErr != nil {
		r.logger.Error("Failed to decrement message reaction count",
			logger.String("message_id", messageID.String()),
			logger.Error(updateErr),
		)
	}

	return nil
}

// GetMessageReactions retrieves all reactions for a specific message, ordered by creation time.
func (r *reactionRepository) GetMessageReactions(ctx context.Context, messageID uuid.UUID) ([]dbModel.Reaction, pkgErrors.AppError) {
	query := `
		SELECT 
			id, message_id, user_id, reaction_type, 
			reaction_emoji, reaction_skin_tone, created_at
		FROM messages.reactions
		WHERE message_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		r.logger.Error("Failed to fetch message reactions",
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to fetch message reactions")
	}
	defer rows.Close()

	var reactions []dbModel.Reaction
	for rows.Next() {
		var reaction dbModel.Reaction
		if scanErr := rows.Scan(
			&reaction.ID,
			&reaction.MessageID,
			&reaction.UserID,
			&reaction.ReactionType,
			&reaction.ReactionEmoji,
			&reaction.ReactionSkinTone,
			&reaction.CreatedAt,
		); scanErr != nil {
			r.logger.Error("Failed to scan reaction row",
				logger.String("message_id", messageID.String()),
				logger.Error(scanErr),
			)
			return nil, pkgErrors.FromError(scanErr, pkgErrors.CodeDatabaseError, "Failed to scan reaction row")
		}
		reactions = append(reactions, reaction)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Error iterating over reaction rows")
	}

	return reactions, nil
}
