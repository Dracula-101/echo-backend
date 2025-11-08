package repo

import (
	"context"
	"fmt"
	"media-service/internal/model"

	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

type FileRepository struct {
	db  database.Database
	log logger.Logger
}

func NewFileRepository(db database.Database, log logger.Logger) *FileRepository {
	return &FileRepository{db: db, log: log}
}

func (r *FileRepository) CreateFile(ctx context.Context, model models.MediaFile) (string, error) {
	var fileID string
	id, err := r.db.Create(ctx, &model)

	if err != nil {
		r.log.Error("Failed to create file record", logger.Error(err))
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	r.log.Info("File record created", logger.String("file_id", id))
	return fileID, nil
}

func (r *FileRepository) GetFileByID(ctx context.Context, fileID string) (*models.MediaFile, error) {
	var model models.MediaFile
	err := r.db.FindByID(ctx, &model, fileID)

	if err != nil {
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get file", logger.String("file_id", fileID), logger.Error(err))
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return &model, nil
}

func (r *FileRepository) ListFilesByUser(ctx context.Context, userID string, limit, offset int) ([]*models.MediaFile, error) {
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
		r.log.Error("Failed to list files", logger.String("user_id", userID), logger.Error(err))
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []*models.MediaFile
	for rows.Next() {
		file := &models.MediaFile{}
		if err := rows.Scan(
			&file.ID, &file.UploaderUserID, &file.FileName, &file.FileType,
			&file.FileSizeBytes, &file.ThumbnailURL, &file.StorageURL,
			&file.ProcessingStatus, &file.Visibility, &file.CreatedAt,
		); err != nil {
			r.log.Error("Failed to scan file row", logger.Error(err))
			continue
		}
		files = append(files, file)
	}

	return files, nil
}

func (r *FileRepository) UpdateFileProcessingStatus(ctx context.Context, fileID, status string, errorMsg string) error {
	query := `
		UPDATE media.files
		SET processing_status = $2,
		    processing_error = $3,
		    processing_completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE processing_completed_at END,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, fileID, status, errorMsg)
	if err != nil {
		r.log.Error("Failed to update file processing status",
			logger.String("file_id", fileID),
			logger.Error(err),
		)
		return fmt.Errorf("failed to update processing status: %w", err)
	}

	return nil
}

func (r *FileRepository) IncrementDownloadCount(ctx context.Context, fileID string) error {
	query := `
		UPDATE media.files
		SET download_count = download_count + 1,
		    last_accessed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		r.log.Error("Failed to increment download count",
			logger.String("file_id", fileID),
			logger.Error(err),
		)
		return fmt.Errorf("failed to increment download count: %w", err)
	}

	return nil
}

func (r *FileRepository) IncrementViewCount(ctx context.Context, fileID string) error {
	query := `
		UPDATE media.files
		SET view_count = view_count + 1,
		    last_accessed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		r.log.Error("Failed to increment view count",
			logger.String("file_id", fileID),
			logger.Error(err),
		)
		return fmt.Errorf("failed to increment view count: %w", err)
	}

	return nil
}

func (r *FileRepository) SoftDeleteFile(ctx context.Context, fileID string) error {
	query := `
		UPDATE media.files
		SET deleted_at = NOW(),
		    permanently_delete_at = NOW() + INTERVAL '30 days',
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		r.log.Error("Failed to soft delete file",
			logger.String("file_id", fileID),
			logger.Error(err),
		)
		return fmt.Errorf("failed to soft delete file: %w", err)
	}

	return nil
}

func (r *FileRepository) HardDeleteFile(ctx context.Context, fileID string) error {
	query := `DELETE FROM media.files WHERE id = $1`

	_, err := r.db.Exec(ctx, query, fileID)
	if err != nil {
		r.log.Error("Failed to hard delete file",
			logger.String("file_id", fileID),
			logger.Error(err),
		)
		return fmt.Errorf("failed to hard delete file: %w", err)
	}

	return nil
}

func (r *FileRepository) GetFileByContentHash(ctx context.Context, contentHash string) (*model.File, error) {
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
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get file by content hash", logger.Error(err))
		return nil, fmt.Errorf("failed to get file by hash: %w", err)
	}

	return file, nil
}

func (r *FileRepository) CountFilesByUser(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM media.files WHERE uploader_user_id = $1 AND deleted_at IS NULL`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		r.log.Error("Failed to count files", logger.String("user_id", userID), logger.Error(err))
		return 0, fmt.Errorf("failed to count files: %w", err)
	}

	return count, nil
}

func (r *FileRepository) GetUserStorageUsage(ctx context.Context, userID string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(file_size_bytes), 0)
		FROM media.files
		WHERE uploader_user_id = $1 AND deleted_at IS NULL
	`

	var totalBytes int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&totalBytes)
	if err != nil {
		r.log.Error("Failed to get storage usage", logger.String("user_id", userID), logger.Error(err))
		return 0, fmt.Errorf("failed to get storage usage: %w", err)
	}

	return totalBytes, nil
}

func (r *FileRepository) CreateAccessLog(ctx context.Context, fileID, userID, accessType, ipAddress, userAgent, deviceID string, success bool, bytesTransferred int64) error {
	query := `
		INSERT INTO media.access_log (
			file_id, user_id, access_type, ip_address, user_agent, device_id,
			success, bytes_transferred
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(ctx, query, fileID, userID, accessType, ipAddress, userAgent, deviceID, success, bytesTransferred)
	if err != nil {
		r.log.Error("Failed to create access log", logger.Error(err))
		return fmt.Errorf("failed to create access log: %w", err)
	}

	return nil
}

func (r *FileRepository) CreateAlbum(ctx context.Context, album *models.Album) (string, error) {
	id, err := r.db.Create(ctx, album)
	if err != nil {
		r.log.Error("Failed to create album", logger.Error(err))
		return "", fmt.Errorf("failed to create album: %w", err)
	}
	return id, nil
}

func (r *FileRepository) GetAlbumByID(ctx context.Context, albumID string) (*models.Album, error) {
	var album models.Album
	err := r.db.FindByID(ctx, &album, albumID)
	if err != nil {
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get album", logger.String("album_id", albumID), logger.Error(err))
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	return &album, nil
}

func (r *FileRepository) ListAlbumsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.Album, error) {
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
		r.log.Error("Failed to list albums", logger.String("user_id", userID), logger.Error(err))
		return nil, fmt.Errorf("failed to list albums: %w", err)
	}
	defer rows.Close()

	var albums []*models.Album
	for rows.Next() {
		album := &models.Album{}
		if err := rows.Scan(
			&album.ID, &album.UserID, &album.Title, &album.Description,
			&album.CoverFileID, &album.AlbumType, &album.IsSystemAlbum,
			&album.FileCount, &album.Visibility, &album.SortOrder,
			&album.CreatedAt, &album.UpdatedAt,
		); err != nil {
			r.log.Error("Failed to scan album row", logger.Error(err))
			continue
		}
		albums = append(albums, album)
	}

	return albums, nil
}

func (r *FileRepository) UpdateAlbum(ctx context.Context, album *models.Album) error {
	err := r.db.Update(ctx, album)
	if err != nil {
		r.log.Error("Failed to update album", logger.String("album_id", album.ID), logger.Error(err))
		return fmt.Errorf("failed to update album: %w", err)
	}
	return nil
}

func (r *FileRepository) DeleteAlbum(ctx context.Context, albumID string) error {
	query := `DELETE FROM media.albums WHERE id = $1`
	_, err := r.db.Exec(ctx, query, albumID)
	if err != nil {
		r.log.Error("Failed to delete album", logger.String("album_id", albumID), logger.Error(err))
		return fmt.Errorf("failed to delete album: %w", err)
	}
	return nil
}

func (r *FileRepository) AddFileToAlbum(ctx context.Context, albumFile *models.AlbumFile) error {
	_, err := r.db.Create(ctx, albumFile)
	if err != nil {
		r.log.Error("Failed to add file to album", logger.Error(err))
		return fmt.Errorf("failed to add file to album: %w", err)
	}
	return nil
}

func (r *FileRepository) RemoveFileFromAlbum(ctx context.Context, albumID, fileID string) error {
	query := `DELETE FROM media.album_files WHERE album_id = $1 AND file_id = $2`
	_, err := r.db.Exec(ctx, query, albumID, fileID)
	if err != nil {
		r.log.Error("Failed to remove file from album", logger.Error(err))
		return fmt.Errorf("failed to remove file from album: %w", err)
	}
	return nil
}

func (r *FileRepository) ListAlbumFiles(ctx context.Context, albumID string, limit, offset int) ([]*models.MediaFile, error) {
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
		r.log.Error("Failed to list album files", logger.String("album_id", albumID), logger.Error(err))
		return nil, fmt.Errorf("failed to list album files: %w", err)
	}
	defer rows.Close()

	var files []*models.MediaFile
	for rows.Next() {
		file := &models.MediaFile{}
		if err := rows.Scan(
			&file.ID, &file.UploaderUserID, &file.FileName, &file.FileType,
			&file.FileSizeBytes, &file.ThumbnailURL, &file.StorageURL,
			&file.ProcessingStatus, &file.Visibility, &file.CreatedAt,
		); err != nil {
			r.log.Error("Failed to scan file row", logger.Error(err))
			continue
		}
		files = append(files, file)
	}

	return files, nil
}

func (r *FileRepository) CreateShare(ctx context.Context, share *models.Share) (string, error) {
	id, err := r.db.Create(ctx, share)
	if err != nil {
		r.log.Error("Failed to create share", logger.Error(err))
		return "", fmt.Errorf("failed to create share: %w", err)
	}
	return id, nil
}

func (r *FileRepository) GetShareByID(ctx context.Context, shareID string) (*models.Share, error) {
	var share models.Share
	err := r.db.FindByID(ctx, &share, shareID)
	if err != nil {
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get share", logger.String("share_id", shareID), logger.Error(err))
		return nil, fmt.Errorf("failed to get share: %w", err)
	}
	return &share, nil
}

func (r *FileRepository) GetShareByToken(ctx context.Context, token string) (*models.Share, error) {
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
		&share.DownloadCount, &share.IsActive, &share.CreatedAt, &share.RevokedAt,
	)

	if err != nil {
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get share by token", logger.Error(err))
		return nil, fmt.Errorf("failed to get share by token: %w", err)
	}

	return &share, nil
}

func (r *FileRepository) RevokeShare(ctx context.Context, shareID string) error {
	query := `
		UPDATE media.shares
		SET is_active = false, revoked_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		r.log.Error("Failed to revoke share", logger.String("share_id", shareID), logger.Error(err))
		return fmt.Errorf("failed to revoke share: %w", err)
	}
	return nil
}

func (r *FileRepository) IncrementShareViewCount(ctx context.Context, shareID string) error {
	query := `
		UPDATE media.shares
		SET view_count = view_count + 1
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		r.log.Error("Failed to increment share view count", logger.String("share_id", shareID), logger.Error(err))
		return fmt.Errorf("failed to increment share view count: %w", err)
	}
	return nil
}

func (r *FileRepository) IncrementShareDownloadCount(ctx context.Context, shareID string) error {
	query := `
		UPDATE media.shares
		SET download_count = download_count + 1
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		r.log.Error("Failed to increment share download count", logger.String("share_id", shareID), logger.Error(err))
		return fmt.Errorf("failed to increment share download count: %w", err)
	}
	return nil
}

func (r *FileRepository) GetStorageStats(ctx context.Context, userID string) (*models.StorageStat, error) {
	var stats models.StorageStat
	query := `SELECT * FROM media.storage_stats WHERE user_id = $1`

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.ID, &stats.UserID, &stats.TotalFiles, &stats.TotalSizeBytes,
		&stats.ImagesCount, &stats.ImagesSizeBytes, &stats.VideosCount, &stats.VideosSizeBytes,
		&stats.AudioCount, &stats.AudioSizeBytes, &stats.DocumentsCount, &stats.DocumentsSizeBytes,
		&stats.StorageQuotaBytes, &stats.StorageUsedPercentage, &stats.LastCalculatedAt,
		&stats.CreatedAt, &stats.UpdatedAt,
	)

	if err != nil {
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get storage stats", logger.String("user_id", userID), logger.Error(err))
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	return &stats, nil
}

func (r *FileRepository) CreateOrUpdateStorageStats(ctx context.Context, stats *models.StorageStat) error {
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
		r.log.Error("Failed to create/update storage stats", logger.Error(err))
		return fmt.Errorf("failed to create/update storage stats: %w", err)
	}

	return nil
}

func (r *FileRepository) CalculateStorageStats(ctx context.Context, userID string) (*models.StorageStat, error) {
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
		r.log.Error("Failed to calculate storage stats", logger.String("user_id", userID), logger.Error(err))
		return nil, fmt.Errorf("failed to calculate storage stats: %w", err)
	}

	stats.UserID = userID

	if stats.StorageQuotaBytes > 0 {
		stats.StorageUsedPercentage = (float64(stats.TotalSizeBytes) / float64(stats.StorageQuotaBytes)) * 100
	}

	return &stats, nil
}
