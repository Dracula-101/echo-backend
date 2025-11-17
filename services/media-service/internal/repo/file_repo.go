package repo

import (
	"context"
	"media-service/internal/model"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type FileRepository struct {
	db  database.Database
	log logger.Logger
}

func NewFileRepository(db database.Database, log logger.Logger) *FileRepository {
	return &FileRepository{db: db, log: log}
}

func (r *FileRepository) CreateFile(ctx context.Context, model models.MediaFile) (string, pkgErrors.AppError) {
	id, err := r.db.Insert(ctx, &model)
	if err != nil {
		return "", pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create file").
			WithDetail("file_name", model.FileName).
			WithDetail("uploader_user_id", model.UploaderUserID)
	}

	return *id, nil
}

func (r *FileRepository) GetFileByID(ctx context.Context, fileID string) (*models.MediaFile, pkgErrors.AppError) {
	var model models.MediaFile
	err := r.db.FindByID(ctx, &model, fileID)
	if err != nil {
		if postgres.IsNoRowsError(err) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "file not found").
				WithDetail("file_id", fileID)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get file").
			WithDetail("file_id", fileID)
	}

	return &model, nil
}

func (r *FileRepository) ListFilesByUser(ctx context.Context, userID string, limit, offset int) ([]*models.MediaFile, pkgErrors.AppError) {
	query := `
		SELECT
			id, uploader_user_id, file_name, file_type, file_size_bytes,
			thumbnail_url, storage_url, processing_status, visibility,
			created_at
		FROM media.files
		WHERE uploader_user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to list files").
			WithDetail("user_id", userID).
			WithDetail("limit", limit).
			WithDetail("offset", offset)
	}
	defer rows.Close()

	var files []*models.MediaFile
	for rows.Next() {
		file := &models.MediaFile{}
		if err := rows.Scan(
			&file.ID, &file.UploaderUserID, &file.FileName, &file.FileType,
			&file.FileSizeBytes, &file.ThumbnailURL, &file.StorageURL,
			&file.ProcessingStatus, &file.Visibility,
		); err != nil {
			// Log scan errors but continue (partial success)
			r.log.Debug("Failed to scan file row", logger.Error(err))
			continue
		}
		files = append(files, file)
	}

	return files, nil
}

func (r *FileRepository) UpdateFileProcessingStatus(ctx context.Context, fileID, status string, errorMsg string) pkgErrors.AppError {
	query := `
		UPDATE media.files
		SET processing_status = $2::text,
		    processing_error = $3::text,
			processing_started_at = CASE WHEN $2::text = 'processing' THEN NOW() ELSE processing_started_at END,
		    processing_completed_at = CASE WHEN $2::text = 'completed' THEN NOW() ELSE processing_completed_at END,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, fileID, status, errorMsg)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update processing status").
			WithDetail("file_id", fileID).
			WithDetail("status", status)
	}

	return nil
}

func (r *FileRepository) UpdateImageMetadata(ctx context.Context, fileID string, width, height int, aspectRatio string, thumbnailSmallURL, thumbnailMediumURL, thumbnailLargeURL *string) pkgErrors.AppError {
	query := `
		UPDATE media.files
		SET width = $2,
		    height = $3,
		    aspect_ratio = $4::text,
		    has_thumbnail = $5,
		    thumbnail_small_url = $6::text,
		    thumbnail_medium_url = $7::text,
		    thumbnail_large_url = $8::text,
		    thumbnail_url = $7::text,
		    updated_at = NOW()
		WHERE id = $1
	`

	hasThumbnail := thumbnailSmallURL != nil || thumbnailMediumURL != nil || thumbnailLargeURL != nil

	_, err := r.db.Exec(ctx, query, fileID, width, height, aspectRatio, hasThumbnail, thumbnailSmallURL, thumbnailMediumURL, thumbnailLargeURL)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update image metadata").
			WithDetail("file_id", fileID)
	}

	return nil
}

func (r *FileRepository) IncrementDownloadCount(ctx context.Context, fileID string) pkgErrors.AppError {
	query := `
		UPDATE media.files
		SET download_count = download_count + 1,
		    last_accessed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to increment download count").
			WithDetail("file_id", fileID)
	}

	return nil
}

func (r *FileRepository) IncrementViewCount(ctx context.Context, fileID string) pkgErrors.AppError {
	query := `
		UPDATE media.files
		SET view_count = view_count + 1,
		    last_accessed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to increment view count").
			WithDetail("file_id", fileID)
	}

	return nil
}

func (r *FileRepository) SoftDeleteFile(ctx context.Context, fileID string) pkgErrors.AppError {
	query := `
		UPDATE media.files
		SET deleted_at = NOW(),
		    permanently_delete_at = NOW() + INTERVAL '30 days',
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to soft delete file").
			WithDetail("file_id", fileID)
	}

	return nil
}

func (r *FileRepository) HardDeleteFile(ctx context.Context, fileID string) pkgErrors.AppError {
	query := `DELETE FROM media.files WHERE id = $1`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to hard delete file").
			WithDetail("file_id", fileID)
	}

	return nil
}

func (r *FileRepository) GetFileByContentHash(ctx context.Context, contentHash string) (*model.File, pkgErrors.AppError) {
	query := `
		SELECT id, storage_url, cdn_url, file_size_bytes, file_type
		FROM media.files
		WHERE content_hash = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	file := &model.File{}
	err := r.db.QueryRow(ctx, query, contentHash).Scan(
		&file.ID, &file.StorageURL, &file.CDNURL, &file.FileSizeBytes, &file.FileType,
	)

	if err != nil {
		if postgres.IsNoRowsError(err) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "file not found by content hash").
				WithDetail("content_hash", contentHash)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get file by hash").
			WithDetail("content_hash", contentHash)
	}

	return file, nil
}

func (r *FileRepository) CountFilesByUser(ctx context.Context, userID string) (int, pkgErrors.AppError) {
	query := `SELECT COUNT(*) FROM media.files WHERE uploader_user_id = $1 AND deleted_at IS NULL`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count files").
			WithDetail("user_id", userID)
	}

	return count, nil
}

func (r *FileRepository) GetUserStorageUsage(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	query := `
		SELECT COALESCE(SUM(file_size_bytes), 0)
		FROM media.files
		WHERE uploader_user_id = $1 AND deleted_at IS NULL
	`

	var totalBytes int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&totalBytes)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get storage usage").
			WithDetail("user_id", userID)
	}

	return totalBytes, nil
}

func (r *FileRepository) CreateAccessLog(ctx context.Context, fileID, userID, accessType, ipAddress, userAgent, deviceID string, success bool, bytesTransferred int64) pkgErrors.AppError {
	query := `
		INSERT INTO media.access_log (
			file_id, user_id, access_type, ip_address, user_agent, device_id,
			success, bytes_transferred
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query, fileID, userID, accessType, ipAddress, userAgent, deviceID, success, bytesTransferred)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create access log").
			WithDetail("file_id", fileID).
			WithDetail("user_id", userID).
			WithDetail("access_type", accessType)
	}

	return nil
}

func (r *FileRepository) CreateAlbum(ctx context.Context, album *models.Album) (string, pkgErrors.AppError) {
	id, err := r.db.Insert(ctx, album)
	if err != nil {
		return "", pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create album").
			WithDetail("user_id", album.UserID).
			WithDetail("title", album.Title)
	}
	return *id, nil
}

func (r *FileRepository) GetAlbumByID(ctx context.Context, albumID string) (*models.Album, pkgErrors.AppError) {
	var album models.Album
	err := r.db.FindByID(ctx, &album, albumID)
	if err != nil {
		if postgres.IsNoRowsError(err) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "album not found").
				WithDetail("album_id", albumID)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get album").
			WithDetail("album_id", albumID)
	}
	return &album, nil
}

func (r *FileRepository) ListAlbumsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.Album, pkgErrors.AppError) {
	query := `
		SELECT id, user_id, title, description, cover_file_id, album_type,
		       is_system_album, file_count, visibility, sort_order, created_at, updated_at
		FROM media.albums
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to list albums").
			WithDetail("user_id", userID).
			WithDetail("limit", limit).
			WithDetail("offset", offset)
	}
	defer rows.Close()

	var albums []*models.Album
	for rows.Next() {
		album := &models.Album{}
		if err := rows.Scan(
			&album.ID, &album.UserID, &album.Title, &album.Description,
			&album.CoverFileID, &album.AlbumType, &album.IsSystemAlbum,
			&album.FileCount, &album.Visibility, &album.SortOrder, &album.UpdatedAt,
		); err != nil {
			// Log scan errors but continue (partial success)
			r.log.Debug("Failed to scan album row", logger.Error(err))
			continue
		}
		albums = append(albums, album)
	}

	return albums, nil
}

func (r *FileRepository) UpdateAlbum(ctx context.Context, album *models.Album) pkgErrors.AppError {
	err := r.db.Update(ctx, album)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update album").
			WithDetail("album_id", album.ID)
	}
	return nil
}

func (r *FileRepository) DeleteAlbum(ctx context.Context, albumID string) pkgErrors.AppError {
	query := `DELETE FROM media.albums WHERE id = $1`
	_, err := r.db.Exec(ctx, query, albumID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete album").
			WithDetail("album_id", albumID)
	}
	return nil
}

func (r *FileRepository) AddFileToAlbum(ctx context.Context, albumFile *models.AlbumFile) pkgErrors.AppError {
	_, err := r.db.Insert(ctx, albumFile)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to add file to album").
			WithDetail("album_id", albumFile.AlbumID).
			WithDetail("file_id", albumFile.FileID)
	}
	return nil
}

func (r *FileRepository) RemoveFileFromAlbum(ctx context.Context, albumID, fileID string) pkgErrors.AppError {
	query := `DELETE FROM media.album_files WHERE album_id = $1 AND file_id = $2`
	_, err := r.db.Exec(ctx, query, albumID, fileID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to remove file from album").
			WithDetail("album_id", albumID).
			WithDetail("file_id", fileID)
	}
	return nil
}

func (r *FileRepository) ListAlbumFiles(ctx context.Context, albumID string, limit, offset int) ([]*models.MediaFile, pkgErrors.AppError) {
	query := `
		SELECT f.id, f.uploader_user_id, f.file_name, f.file_type, f.file_size_bytes,
		       f.thumbnail_url, f.storage_url, f.processing_status, f.visibility, f.created_at
		FROM media.files f
		INNER JOIN media.album_files af ON f.id = af.file_id
		WHERE af.album_id = $1
		ORDER BY af.display_order ASC, af.added_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, albumID, limit, offset)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to list album files").
			WithDetail("album_id", albumID).
			WithDetail("limit", limit).
			WithDetail("offset", offset)
	}
	defer rows.Close()

	var files []*models.MediaFile
	for rows.Next() {
		file := &models.MediaFile{}
		if err := rows.Scan(
			&file.ID, &file.UploaderUserID, &file.FileName, &file.FileType,
			&file.FileSizeBytes, &file.ThumbnailURL, &file.StorageURL,
			&file.ProcessingStatus, &file.Visibility,
		); err != nil {
			// Log scan errors but continue (partial success)
			r.log.Debug("Failed to scan file row", logger.Error(err))
			continue
		}
		files = append(files, file)
	}

	return files, nil
}

func (r *FileRepository) CreateShare(ctx context.Context, share *models.Share) (string, pkgErrors.AppError) {
	id, err := r.db.Insert(ctx, share)
	if err != nil {
		return "", pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create share").
			WithDetail("file_id", share.FileID).
			WithDetail("shared_by_user_id", share.SharedByUserID)
	}
	return *id, nil
}

func (r *FileRepository) GetShareByID(ctx context.Context, shareID string) (*models.Share, pkgErrors.AppError) {
	var share models.Share
	err := r.db.FindByID(ctx, &share, shareID)
	if err != nil {
		if postgres.IsNoRowsError(err) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "share not found").
				WithDetail("share_id", shareID)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get share").
			WithDetail("share_id", shareID)
	}
	return &share, nil
}

func (r *FileRepository) GetShareByToken(ctx context.Context, token string) (*models.Share, pkgErrors.AppError) {
	query := `
		SELECT id, file_id, shared_by_user_id, shared_with_user_id,
		       shared_with_conversation_id, share_token, access_type,
		       password_hash, expires_at, max_views, view_count,
		       download_count, is_active, created_at, revoked_at
		FROM media.shares
		WHERE share_token = $1 AND is_active = true
	`

	var share models.Share
	err := r.db.QueryRow(ctx, query, token).Scan(
		&share.ID, &share.FileID, &share.SharedByUserID, &share.SharedWithUserID,
		&share.SharedWithConversationID, &share.ShareToken, &share.AccessType,
		&share.PasswordHash, &share.ExpiresAt, &share.MaxViews, &share.ViewCount,
		&share.DownloadCount, &share.IsActive, &share.RevokedAt,
	)

	if err != nil {
		if postgres.IsNoRowsError(err) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "share not found by token").
				WithDetail("share_token", token)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get share by token").
			WithDetail("share_token", token)
	}

	return &share, nil
}

func (r *FileRepository) RevokeShare(ctx context.Context, shareID string) pkgErrors.AppError {
	query := `
		UPDATE media.shares
		SET is_active = false, revoked_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke share").
			WithDetail("share_id", shareID)
	}
	return nil
}

func (r *FileRepository) IncrementShareViewCount(ctx context.Context, shareID string) pkgErrors.AppError {
	query := `
		UPDATE media.shares
		SET view_count = view_count + 1
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to increment share view count").
			WithDetail("share_id", shareID)
	}
	return nil
}

func (r *FileRepository) IncrementShareDownloadCount(ctx context.Context, shareID string) pkgErrors.AppError {
	query := `
		UPDATE media.shares
		SET download_count = download_count + 1
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to increment share download count").
			WithDetail("share_id", shareID)
	}
	return nil
}

func (r *FileRepository) GetStorageStats(ctx context.Context, userID string) (*models.StorageStat, pkgErrors.AppError) {
	var stats models.StorageStat
	query := `SELECT * FROM media.storage_stats WHERE user_id = $1`

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.ID, &stats.UserID, &stats.TotalFiles, &stats.TotalSizeBytes,
		&stats.ImagesCount, &stats.ImagesSizeBytes, &stats.VideosCount, &stats.VideosSizeBytes,
		&stats.AudioCount, &stats.AudioSizeBytes, &stats.DocumentsCount, &stats.DocumentsSizeBytes,
		&stats.StorageQuotaBytes, &stats.StorageUsedPercentage, &stats.LastCalculatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		if postgres.IsNoRowsError(err) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "storage stats not found").
				WithDetail("user_id", userID)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get storage stats").
			WithDetail("user_id", userID)
	}

	return &stats, nil
}

func (r *FileRepository) CreateOrUpdateStorageStats(ctx context.Context, stats *models.StorageStat) pkgErrors.AppError {
	query := `
		INSERT INTO media.storage_stats (
			user_id, total_files, total_size_bytes, images_count, images_size_bytes,
			videos_count, videos_size_bytes, audio_count, audio_size_bytes,
			documents_count, documents_size_bytes, storage_quota_bytes,
			storage_used_percentage, last_calculated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			total_files = EXCLUDED.total_files,
			total_size_bytes = EXCLUDED.total_size_bytes,
			images_count = EXCLUDED.images_count,
			images_size_bytes = EXCLUDED.images_size_bytes,
			videos_count = EXCLUDED.videos_count,
			videos_size_bytes = EXCLUDED.videos_size_bytes,
			audio_count = EXCLUDED.audio_count,
			audio_size_bytes = EXCLUDED.audio_size_bytes,
			documents_count = EXCLUDED.documents_count,
			documents_size_bytes = EXCLUDED.documents_size_bytes,
			storage_quota_bytes = EXCLUDED.storage_quota_bytes,
			storage_used_percentage = EXCLUDED.storage_used_percentage,
			last_calculated_at = NOW(),
			updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query,
		stats.UserID, stats.TotalFiles, stats.TotalSizeBytes,
		stats.ImagesCount, stats.ImagesSizeBytes,
		stats.VideosCount, stats.VideosSizeBytes,
		stats.AudioCount, stats.AudioSizeBytes,
		stats.DocumentsCount, stats.DocumentsSizeBytes,
		stats.StorageQuotaBytes, stats.StorageUsedPercentage,
	)

	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create/update storage stats").
			WithDetail("user_id", stats.UserID)
	}

	return nil
}

func (r *FileRepository) CalculateStorageStats(ctx context.Context, userID string) (*models.StorageStat, pkgErrors.AppError) {
	query := `
		SELECT COUNT(*) as total_files,
			COALESCE(SUM(file_size_bytes), 0) as total_size_bytes,
			COALESCE(SUM(CASE WHEN file_category = 'image' THEN 1 ELSE 0 END), 0) as images_count,
			COALESCE(SUM(CASE WHEN file_category = 'image' THEN file_size_bytes ELSE 0 END), 0) as images_size_bytes,
			COALESCE(SUM(CASE WHEN file_category = 'video' THEN 1 ELSE 0 END), 0) as videos_count,
			COALESCE(SUM(CASE WHEN file_category = 'video' THEN file_size_bytes ELSE 0 END), 0) as videos_size_bytes,
			COALESCE(SUM(CASE WHEN file_category = 'audio' THEN 1 ELSE 0 END), 0) as audio_count,
			COALESCE(SUM(CASE WHEN file_category = 'audio' THEN file_size_bytes ELSE 0 END), 0) as audio_size_bytes,
			COALESCE(SUM(CASE WHEN file_category = 'document' THEN 1 ELSE 0 END), 0) as documents_count,
			COALESCE(SUM(CASE WHEN file_category = 'document' THEN file_size_bytes ELSE 0 END), 0) as documents_size_bytes
		FROM media.files WHERE uploader_user_id = $1 AND deleted_at IS NULL
	`

	var stats models.StorageStat
	stats.StorageQuotaBytes = 5368709120

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.TotalFiles, &stats.TotalSizeBytes,
		&stats.ImagesCount, &stats.ImagesSizeBytes,
		&stats.VideosCount, &stats.VideosSizeBytes,
		&stats.AudioCount, &stats.AudioSizeBytes,
		&stats.DocumentsCount, &stats.DocumentsSizeBytes,
	)

	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to calculate storage stats").
			WithDetail("user_id", userID)
	}

	stats.UserID = userID

	if stats.StorageQuotaBytes > 0 {
		stats.StorageUsedPercentage = (float64(stats.TotalSizeBytes) / float64(stats.StorageQuotaBytes)) * 100
	}

	return &stats, nil
}
