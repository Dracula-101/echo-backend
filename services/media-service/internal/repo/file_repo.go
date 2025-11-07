package repo

import (
	"context"
	"fmt"

	"media-service/internal/model"

	"shared/pkg/database"
	"shared/pkg/logger"
)

type FileRepository struct {
	db  database.Database
	log logger.Logger
}

func NewFileRepository(db database.Database, log logger.Logger) *FileRepository {
	return &FileRepository{
		db:  db,
		log: log,
	}
}

// CreateFileParams represents parameters for creating a file record
type CreateFileParams struct {
	UploaderUserID   string
	FileName         string
	OriginalFileName string
	FileType         string
	MimeType         string
	FileCategory     string
	FileExtension    string
	FileSizeBytes    int64
	StorageProvider  string
	StorageBucket    string
	StorageKey       string
	StorageURL       string
	StorageRegion    string
	CDNURL           string
	ContentHash      string
	Visibility       string
	DeviceID         string
	IPAddress        string
}

// CreateFile creates a new file record in the database
func (r *FileRepository) CreateFile(ctx context.Context, params CreateFileParams) (string, error) {
	query := `
		INSERT INTO media.files (
			uploader_user_id, file_name, original_file_name, file_type, mime_type,
			file_category, file_extension, file_size_bytes, storage_provider,
			storage_bucket, storage_key, storage_url, storage_region, cdn_url,
			content_hash, visibility, processing_status, uploaded_from_device_id,
			uploaded_from_ip
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			'pending', $17, $18
		) RETURNING id
	`

	var fileID string
	err := r.db.QueryRow(ctx, query,
		params.UploaderUserID, params.FileName, params.OriginalFileName,
		params.FileType, params.MimeType, params.FileCategory, params.FileExtension,
		params.FileSizeBytes, params.StorageProvider, params.StorageBucket,
		params.StorageKey, params.StorageURL, params.StorageRegion, params.CDNURL,
		params.ContentHash, params.Visibility, params.DeviceID, params.IPAddress,
	).Scan(&fileID)

	if err != nil {
		r.log.Error("Failed to create file record", logger.Error(err))
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	r.log.Info("File record created", logger.String("file_id", fileID))
	return fileID, nil
}

// GetFileByID retrieves a file by its ID
func (r *FileRepository) GetFileByID(ctx context.Context, fileID string) (*model.File, error) {
	query := `
		SELECT
			id, uploader_user_id, file_name, original_file_name, file_type, mime_type,
			file_category, file_extension, file_size_bytes, storage_provider,
			storage_bucket, storage_key, storage_url, storage_region, cdn_url,
			has_thumbnail, thumbnail_url, thumbnail_small_url, thumbnail_medium_url,
			thumbnail_large_url, has_preview, preview_url, width, height,
			duration_seconds, aspect_ratio, processing_status, processing_error,
			is_encrypted, content_hash, is_scanned, virus_scan_status, visibility,
			access_token, expires_at, download_count, view_count, uploaded_at,
			created_at, updated_at, deleted_at
		FROM media.files
		WHERE id = $1 AND deleted_at IS NULL
	`

	file := &model.File{}
	err := r.db.QueryRow(ctx, query, fileID).Scan(
		&file.ID, &file.UploaderUserID, &file.FileName, &file.OriginalFileName,
		&file.FileType, &file.MimeType, &file.FileCategory, &file.FileExtension,
		&file.FileSizeBytes, &file.StorageProvider, &file.StorageBucket,
		&file.StorageKey, &file.StorageURL, &file.StorageRegion, &file.CDNURL,
		&file.HasThumbnail, &file.ThumbnailURL, &file.ThumbnailSmallURL,
		&file.ThumbnailMediumURL, &file.ThumbnailLargeURL, &file.HasPreview,
		&file.PreviewURL, &file.Width, &file.Height, &file.DurationSeconds,
		&file.AspectRatio, &file.ProcessingStatus, &file.ProcessingError,
		&file.IsEncrypted, &file.ContentHash, &file.IsScanned, &file.VirusScanStatus,
		&file.Visibility, &file.AccessToken, &file.ExpiresAt, &file.DownloadCount,
		&file.ViewCount, &file.UploadedAt, &file.CreatedAt, &file.UpdatedAt,
		&file.DeletedAt,
	)

	if err != nil {
		if database.IsNoRowsError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get file", logger.String("file_id", fileID), logger.Error(err))
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

// ListFilesByUser retrieves files for a specific user
func (r *FileRepository) ListFilesByUser(ctx context.Context, userID string, limit, offset int) ([]*model.File, error) {
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

	var files []*model.File
	for rows.Next() {
		file := &model.File{}
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

// UpdateFileProcessingStatus updates the processing status of a file
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

// IncrementDownloadCount increments the download count for a file
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

// IncrementViewCount increments the view count for a file
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

// SoftDeleteFile soft deletes a file
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

// HardDeleteFile permanently deletes a file
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

// GetFileByContentHash retrieves a file by its content hash (for deduplication)
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

// CountFilesByUser counts total files for a user
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

// GetUserStorageUsage retrieves storage usage for a user
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

// CreateAccessLog creates an access log entry
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
