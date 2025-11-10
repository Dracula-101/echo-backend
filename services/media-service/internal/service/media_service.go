package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"media-service/internal/config"
	mediaErrors "media-service/internal/errors"
	"media-service/internal/model"
	"media-service/internal/repo"
	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/media"
	"shared/pkg/utils"

	"shared/pkg/cache"
	"shared/pkg/logger"

	"github.com/disintegration/imaging"
)

type MediaService struct {
	repo            *repo.FileRepository
	cache           cache.Cache
	cfg             *config.Config
	log             logger.Logger
	storageProvider StorageProvider
}

type MediaServiceBuilder struct {
	fileRepo        *repo.FileRepository
	cache           cache.Cache
	cfg             *config.Config
	log             logger.Logger
	storageProvider StorageProvider
}

type StorageProvider interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

func NewMediaServiceBuilder() *MediaServiceBuilder {
	return &MediaServiceBuilder{}
}

func (b *MediaServiceBuilder) WithFileRepo(r *repo.FileRepository) *MediaServiceBuilder {
	b.fileRepo = r
	return b
}

func (b *MediaServiceBuilder) WithCache(c cache.Cache) *MediaServiceBuilder {
	b.cache = c
	return b
}

func (b *MediaServiceBuilder) WithConfig(c *config.Config) *MediaServiceBuilder {
	b.cfg = c
	return b
}

func (b *MediaServiceBuilder) WithLogger(l logger.Logger) *MediaServiceBuilder {
	b.log = l
	return b
}

func (b *MediaServiceBuilder) WithStorageProvider(p StorageProvider) *MediaServiceBuilder {
	b.storageProvider = p
	return b
}

func (b *MediaServiceBuilder) Build() *MediaService {
	if b.fileRepo == nil {
		panic("FileRepository is required")
	}
	if b.cfg == nil {
		panic("Config is required")
	}
	if b.log == nil {
		panic("Logger is required")
	}
	if b.storageProvider == nil {
		panic("StorageProvider is required")
	}

	b.log.Info("Building MediaService", logger.String("service", mediaErrors.ServiceName))

	return &MediaService{
		repo:            b.fileRepo,
		cache:           b.cache,
		cfg:             b.cfg,
		log:             b.log,
		storageProvider: b.storageProvider,
	}
}

func (s *MediaService) UploadFile(ctx context.Context, input models.UploadFileInput) (*models.UploadFileOutput, pkgErrors.AppError) {
	s.log.Info("Uploading file",
		logger.String("user_id", input.UserID),
		logger.String("file_name", input.FileName),
		logger.Int64("file_size", input.FileSize),
	)

	if input.FileSize > s.cfg.Storage.MaxFileSize {
		s.log.Warn("File too large",
			logger.String("user_id", input.UserID),
			logger.Int64("file_size", input.FileSize),
			logger.Int64("max_size", s.cfg.Storage.MaxFileSize),
		)
		return nil, pkgErrors.New(mediaErrors.CodeFileTooLarge, "file size exceeds the maximum allowed").
			WithDetail("max_size_bytes", s.cfg.Storage.MaxFileSize).
			WithDetail("file_size_bytes", input.FileSize).
			WithService("media-service")
	}

	storageUsed, err := s.repo.GetUserStorageUsage(ctx, input.UserID)
	if err != nil {
		s.log.Error("Failed to check storage quota", logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to check storage quota").
			WithDetail("user_id", input.UserID).
			WithService("media-service")
	}

	storageQuota := int64(5 * 1024 * 1024 * 1024)
	if storageUsed+input.FileSize > storageQuota {
		s.log.Warn("Storage quota exceeded",
			logger.String("user_id", input.UserID),
			logger.Int64("current_usage", storageUsed),
			logger.Int64("quota", storageQuota),
		)
		return nil, pkgErrors.New(mediaErrors.CodeStorageQuotaExceeded, "storage quota exceeded").
			WithDetail("current_usage", storageUsed).
			WithDetail("quota", storageQuota).
			WithService("media-service")
	}

	var buf bytes.Buffer
	teeReader := io.TeeReader(input.FileReader, &buf)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, teeReader); err != nil {
		s.log.Error("Failed to read file", logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to read file").
			WithDetail("file_name", input.FileName).
			WithService("media-service")
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	if s.cfg.Features.Deduplication.Enabled {
		existingFile, err := s.repo.GetFileByContentHash(ctx, contentHash)
		if err == nil && existingFile != nil {
			s.log.Info("Duplicate file detected, reusing existing",
				logger.String("existing_file_id", existingFile.ID))
			return &models.UploadFileOutput{
				FileID:           existingFile.ID,
				FileName:         input.FileName,
				FileSize:         existingFile.FileSizeBytes,
				FileType:         existingFile.FileType,
				StorageURL:       existingFile.StorageURL,
				CDNURL:           existingFile.CDNURL,
				ProcessingStatus: existingFile.ProcessingStatus,
				UploadedAt:       time.Now(),
			}, nil
		}
	}

	fileExt := filepath.Ext(input.FileName)
	storageKey := fmt.Sprintf("%s-%s-%s%s", input.UserID, time.Now().Format("2006-01-02"), contentHash, fileExt)

	storageURL, uploadErr := s.storageProvider.Upload(ctx, storageKey, &buf, input.ContentType)
	if uploadErr != nil {
		s.log.Error("Failed to upload to storage", logger.Error(uploadErr))
		return nil, pkgErrors.FromError(uploadErr, pkgErrors.CodeInternal, "failed to upload file").
			WithDetail("storage_key", storageKey).
			WithDetail("content_type", input.ContentType).
			WithService("media-service")
	}

	fileCategory := determineFileCategory(input.ContentType)
	storageRegion := strings.TrimSpace(s.cfg.Storage.Region)

	cdnURL := s.buildCDNURL(storageKey)
	fileID, err := s.repo.CreateFile(ctx, dbModels.MediaFile{
		UploaderUserID:       input.UserID,
		FileName:             input.FileName,
		OriginalFileName:     utils.PtrString(input.FileName),
		FileType:             input.ContentType,
		MimeType:             input.ContentType,
		FileCategory:         utils.PtrString(fileCategory),
		FileExtension:        utils.PtrString(strings.TrimPrefix(fileExt, ".")),
		FileSizeBytes:        input.FileSize,
		StorageProvider:      utils.PtrString(s.cfg.Storage.Provider),
		StorageBucket:        utils.PtrString(s.cfg.Storage.Bucket),
		StorageKey:           storageKey,
		StorageURL:           storageURL,
		StorageRegion:        utils.PtrString(storageRegion),
		CDNURL:               utils.PtrString(cdnURL),
		ContentHash:          utils.PtrString(contentHash),
		ProcessingStatus:     dbModels.FileProcessingStatusPending,
		ModerationStatus:     dbModels.ModerationStatusPending,
		Visibility:           dbModels.MediaVisibility(input.Visibility),
		UploadedFromDeviceID: utils.PtrString(input.DeviceID),
		UploadedFromIP:       utils.PtrString(input.IPAddress),
	})

	if err != nil {
		s.log.Error("Failed to create file record", logger.Error(err))
		_ = s.storageProvider.Delete(ctx, storageKey)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to create file record").
			WithDetail("file_name", input.FileName).
			WithDetail("storage_key", storageKey).
			WithService("media-service")
	}

	s.log.Info("File uploaded successfully", logger.String("file_id", fileID))

	_ = s.repo.CreateAccessLog(ctx, fileID, input.UserID, "upload", input.IPAddress, input.UserAgent, input.DeviceID, true, input.FileSize)

	return &models.UploadFileOutput{
		FileID:           fileID,
		FileName:         input.FileName,
		FileSize:         input.FileSize,
		FileType:         input.ContentType,
		StorageURL:       storageURL,
		CDNURL:           s.buildCDNURL(storageKey),
		ProcessingStatus: "pending",
		UploadedAt:       time.Now(),
	}, nil
}

func (s *MediaService) ProcessImageFile(ctx context.Context, fileID string, processor *media.Processor) error {
	s.log.Info("Processing image file", logger.String("file_id", fileID))

	if err := s.repo.UpdateFileProcessingStatus(ctx, fileID, "processing", ""); err != nil {
		s.log.Error("Failed to update processing status", logger.Error(err))
		return err
	}

	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		s.log.Error("Failed to get file", logger.Error(err))
		_ = s.repo.UpdateFileProcessingStatus(ctx, fileID, "failed", "file not found")
		return err
	}

	if file.FileCategory == nil || *file.FileCategory != "image" {
		s.log.Debug("Skipping non-image file", logger.String("file_id", fileID))
		_ = s.repo.UpdateFileProcessingStatus(ctx, fileID, "completed", "")
		return nil
	}

	reader, downloadErr := s.storageProvider.Download(ctx, file.StorageKey)
	if downloadErr != nil {
		s.log.Error("Failed to download file from storage", logger.Error(downloadErr))
		_ = s.repo.UpdateFileProcessingStatus(ctx, fileID, "failed", "failed to download from storage")
		return downloadErr
	}
	defer reader.Close()

	imageData, readErr := io.ReadAll(reader)
	if readErr != nil {
		s.log.Error("Failed to read image data", logger.Error(readErr))
		_ = s.repo.UpdateFileProcessingStatus(ctx, fileID, "failed", "failed to read image data")
		return readErr
	}

	imgConfig, format, decodeErr := image.DecodeConfig(bytes.NewReader(imageData))
	if decodeErr != nil {
		s.log.Error("Failed to decode image", logger.Error(decodeErr))
		_ = s.repo.UpdateFileProcessingStatus(ctx, fileID, "failed", "invalid image format")
		return decodeErr
	}

	aspectRatio := fmt.Sprintf("%.2f:1", float64(imgConfig.Width)/float64(imgConfig.Height))

	s.log.Info("Image dimensions extracted",
		logger.String("file_id", fileID),
		logger.Int("width", imgConfig.Width),
		logger.Int("height", imgConfig.Height),
		logger.String("format", format),
	)

	var thumbnailSmall, thumbnailMedium, thumbnailLarge *string

	if s.cfg.Features.Thumbnails.Enabled {
		urls, thumbErr := s.generateAndUploadThumbnails(ctx, imageData, file, format)
		if thumbErr != nil {
			s.log.Error("Failed to generate thumbnails", logger.Error(thumbErr))
		} else {
			if url, ok := urls["small"]; ok {
				thumbnailSmall = &url
			}
			if url, ok := urls["medium"]; ok {
				thumbnailMedium = &url
			}
			if url, ok := urls["large"]; ok {
				thumbnailLarge = &url
			}
		}
	}

	if updateErr := s.repo.UpdateImageMetadata(ctx, fileID, imgConfig.Width, imgConfig.Height, aspectRatio, thumbnailSmall, thumbnailMedium, thumbnailLarge); updateErr != nil {
		s.log.Error("Failed to update image metadata", logger.Error(updateErr))
		_ = s.repo.UpdateFileProcessingStatus(ctx, fileID, "failed", "failed to update metadata")
		return updateErr
	}

	if completeErr := s.repo.UpdateFileProcessingStatus(ctx, fileID, "completed", ""); completeErr != nil {
		s.log.Error("Failed to update processing status to completed", logger.Error(completeErr))
		return completeErr
	}

	s.log.Info("Image processing completed successfully", logger.String("file_id", fileID))
	return nil
}

func (s *MediaService) generateAndUploadThumbnails(ctx context.Context, imageData []byte, file *dbModels.MediaFile, format string) (map[string]string, error) {
	img, _, decodeErr := image.Decode(bytes.NewReader(imageData))
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode image for thumbnails: %w", decodeErr)
	}

	// Define thumbnail sizes from config
	thumbnailSizes := map[string]struct{ width, height int }{
		"small":  {s.cfg.Features.Thumbnails.SmallSize.Width, s.cfg.Features.Thumbnails.SmallSize.Height},
		"medium": {s.cfg.Features.Thumbnails.MediumSize.Width, s.cfg.Features.Thumbnails.MediumSize.Height},
		"large":  {s.cfg.Features.Thumbnails.LargeSize.Width, s.cfg.Features.Thumbnails.LargeSize.Height},
	}

	thumbnailURLs := make(map[string]string)

	for sizeName, size := range thumbnailSizes {
		resized := imaging.Fit(img, size.width, size.height, imaging.Lanczos)

		var buf bytes.Buffer
		var encodeErr error

		switch strings.ToLower(format) {
		case "jpeg", "jpg":
			encodeErr = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: s.cfg.Features.ImageProcessing.Quality})
		case "png":
			encodeErr = png.Encode(&buf, resized)
		default:
			encodeErr = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: s.cfg.Features.ImageProcessing.Quality})
		}

		if encodeErr != nil {
			s.log.Error("Failed to encode thumbnail",
				logger.String("size", sizeName),
				logger.Error(encodeErr),
			)
			continue
		}

		// Generate storage path for thumbnail
		thumbnailKey := GenerateThumbnailPath(
			file.StorageKey,
			sizeName,
			*file.ContentHash,
			filepath.Ext(file.FileName),
		)

		// Upload thumbnail to storage
		thumbnailURL, uploadErr := s.storageProvider.Upload(
			ctx,
			thumbnailKey,
			bytes.NewReader(buf.Bytes()),
			file.FileType,
		)

		if uploadErr != nil {
			s.log.Error("Failed to upload thumbnail",
				logger.String("size", sizeName),
				logger.Error(uploadErr),
			)
			continue
		}

		s.log.Debug("Thumbnail uploaded",
			logger.String("size", sizeName),
			logger.String("url", thumbnailURL),
		)

		thumbnailURLs[sizeName] = thumbnailURL
	}

	return thumbnailURLs, nil
}

func (s *MediaService) GetFile(ctx context.Context, input models.GetFileInput) (*models.GetFileOutput, error) {
	s.log.Info("Getting file",
		logger.String("file_id", input.FileID),
		logger.String("user_id", input.UserID),
	)

	fileParam, err := s.repo.GetFileByID(ctx, input.FileID)
	if err != nil {
		s.log.Error("Failed to get file", logger.Error(err))
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	file := &model.File{
		ID:               fileParam.ID,
		UploaderUserID:   fileParam.UploaderUserID,
		FileName:         fileParam.FileName,
		FileSizeBytes:    fileParam.FileSizeBytes,
		FileType:         fileParam.FileType,
		StorageURL:       fileParam.StorageURL,
		CDNURL:           *fileParam.CDNURL,
		ThumbnailURL:     *fileParam.ThumbnailURL,
		ProcessingStatus: fileParam.ProcessingStatus.String(),
		Visibility:       model.VisibilityType(fileParam.Visibility),
		DownloadCount:    fileParam.DownloadCount,
		ViewCount:        fileParam.ViewCount,
		CreatedAt:        fileParam.CreatedAt,
	}

	if file.Visibility == model.VisibilityPrivate && file.UploaderUserID != input.UserID {
		s.log.Warn("Access denied to file",
			logger.String("file_id", input.FileID),
			logger.String("user_id", input.UserID),
		)
		return nil, fmt.Errorf("access denied: %w", fmt.Errorf(mediaErrors.CodeAccessDenied))
	}

	_ = s.repo.IncrementViewCount(ctx, input.FileID)

	return &models.GetFileOutput{
		FileID:           file.ID,
		FileName:         file.FileName,
		FileSize:         file.FileSizeBytes,
		FileType:         file.FileType,
		StorageURL:       file.StorageURL,
		CDNURL:           file.CDNURL,
		ThumbnailURL:     file.ThumbnailURL,
		ProcessingStatus: file.ProcessingStatus,
		Visibility:       file.Visibility.String(),
		DownloadCount:    file.DownloadCount,
		ViewCount:        file.ViewCount,
		CreatedAt:        file.CreatedAt,
	}, nil
}

func (s *MediaService) DeleteFile(ctx context.Context, input models.DeleteFileInput) error {
	s.log.Info("Deleting file",
		logger.String("file_id", input.FileID),
		logger.String("user_id", input.UserID),
		logger.Bool("permanent", input.Permanent),
	)

	f, getErr := s.repo.GetFileByID(ctx, input.FileID)
	if getErr != nil || f == nil {
		return fmt.Errorf("file not found: %w", fmt.Errorf(mediaErrors.CodeFileNotFound))
	}

	if f.UploaderUserID != input.UserID {
		return fmt.Errorf("access denied: %w", fmt.Errorf(mediaErrors.CodeAccessDenied))
	}

	var err error
	if input.Permanent {
		if delErr := s.storageProvider.Delete(ctx, f.StorageKey); delErr != nil {
			s.log.Error("Failed to delete from storage", logger.Error(delErr))
		}

		err = s.repo.HardDeleteFile(ctx, input.FileID)
	} else {
		err = s.repo.SoftDeleteFile(ctx, input.FileID)
	}

	if err != nil {
		s.log.Error("Failed to delete file", logger.Error(err))
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.log.Info("File deleted successfully", logger.String("file_id", input.FileID))
	return nil
}

func (s *MediaService) buildCDNURL(storageKey string) string {
	if s.cfg.Storage.UseCDN && s.cfg.Storage.CDNBaseURL != "" {
		return fmt.Sprintf("%s/%s", s.cfg.Storage.CDNBaseURL, storageKey)
	}
	return ""
}

func determineFileCategory(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "application/pdf"):
		return "document"
	case strings.Contains(contentType, "document"):
		return "document"
	default:
		return "other"
	}
}
