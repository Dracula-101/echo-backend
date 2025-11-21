package repo

import (
	"context"

	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

func (r *FileRepository) ConversationExists(ctx context.Context, conversationID string) (bool, pkgErrors.AppError) {
	query := `SELECT EXISTS(SELECT 1 FROM messages.conversations WHERE id = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRow(ctx, query, conversationID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check conversation existence",
			logger.String("conversation_id", conversationID),
			logger.Error(err))
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check conversation existence").
			WithDetail("conversation_id", conversationID).
			WithService("media_service")
	}

	return exists, nil
}

func (r *FileRepository) UserExists(ctx context.Context, userID string) (bool, pkgErrors.AppError) {
	query := `SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check user existence",
			logger.String("user_id", userID),
			logger.Error(err))
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check user existence").
			WithDetail("user_id", userID).
			WithService("media_service")
	}

	return exists, nil
}

func (r *FileRepository) AlbumExistsAndOwned(ctx context.Context, albumID, userID string) (bool, pkgErrors.AppError) {
	query := `SELECT EXISTS(SELECT 1 FROM media.albums WHERE id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, albumID, userID).Scan(&exists)
	if err != nil {
		if postgres.IsNoRowsError(err) {
			return false, nil
		}
		r.log.Error("Failed to check album ownership",
			logger.String("album_id", albumID),
			logger.String("user_id", userID),
			logger.Error(err))
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check album ownership").
			WithDetail("album_id", albumID).
			WithDetail("user_id", userID).
			WithService("media_service")
	}

	return exists, nil
}

func (r *FileRepository) FileExistsAndOwned(ctx context.Context, fileID, userID string) (bool, pkgErrors.AppError) {
	query := `SELECT EXISTS(SELECT 1 FROM media.files WHERE id = $1 AND uploader_user_id = $2 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRow(ctx, query, fileID, userID).Scan(&exists)
	if err != nil {
		if postgres.IsNoRowsError(err) {
			return false, nil
		}
		r.log.Error("Failed to check file ownership",
			logger.String("file_id", fileID),
			logger.String("user_id", userID),
			logger.Error(err))
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check file ownership").
			WithDetail("file_id", fileID).
			WithDetail("user_id", userID).
			WithService("media_service")
	}

	return exists, nil
}
