package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

	"shared/pkg/cache"
	"shared/pkg/circuitbreaker"
	"shared/pkg/logger"
	"shared/pkg/retry"
)

type StorageProvider interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

type MediaService struct {
	repo            *repo.FileRepository
	cache           cache.Cache
	cfg             *config.Config
	log             logger.Logger
	dbCircuit       *circuitbreaker.CircuitBreaker
	retryer         *retry.Retryer
	storageProvider StorageProvider
}

type MediaServiceBuilder struct {
	fileRepo        *repo.FileRepository
	cache           cache.Cache
	cfg             *config.Config
	log             logger.Logger
	storageProvider StorageProvider
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

	dbCircuit := circuitbreaker.New("media-db", circuitbreaker.Config{
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			b.log.Info("Circuit breaker state changed",
				logger.String("circuit", name),
				logger.String("from", from.String()),
				logger.String("to", to.String()),
			)
		},
	})

	retryer := retry.New(retry.Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Strategy:     retry.StrategyExponential,
		Multiplier:   2.0,
		OnRetry: func(attempt int, delay time.Duration, err error) {
			b.log.Warn("Retrying operation",
				logger.Int("attempt", attempt),
				logger.Duration("delay", delay),
				logger.Error(err),
			)
		},
	})

	return &MediaService{
		repo:            b.fileRepo,
		cache:           b.cache,
		cfg:             b.cfg,
		log:             b.log,
		dbCircuit:       dbCircuit,
		retryer:         retryer,
		storageProvider: b.storageProvider,
	}
}

func (s *MediaService) UploadFile(ctx context.Context, input models.UploadFileInput) (*models.UploadFileOutput, error) {
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
		return nil, fmt.Errorf("file size exceeds maximum allowed: %w", fmt.Errorf(mediaErrors.CodeFileTooLarge))
	}

	var storageUsed int64
	err := s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
			usage, err := s.repo.GetUserStorageUsage(ctx, input.UserID)
			if err != nil {
				return err
			}
			storageUsed = usage
			return nil
		})
	})

	if err != nil {
		s.log.Error("Failed to check storage quota", logger.Error(err))
		return nil, fmt.Errorf("failed to check storage quota: %w", err)
	}

	storageQuota := int64(5 * 1024 * 1024 * 1024)
	if storageUsed+input.FileSize > storageQuota {
		s.log.Warn("Storage quota exceeded",
			logger.String("user_id", input.UserID),
			logger.Int64("current_usage", storageUsed),
			logger.Int64("quota", storageQuota),
		)
		return nil, fmt.Errorf("storage quota exceeded: %w", fmt.Errorf(mediaErrors.CodeStorageQuotaExceeded))
	}

	var buf bytes.Buffer
	teeReader := io.TeeReader(input.FileReader, &buf)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, teeReader); err != nil {
		s.log.Error("Failed to read file", logger.Error(err))
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	if s.cfg.Features.Deduplication.Enabled {
		var existingFile *model.File
		err := s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
			return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
				file, err := s.repo.GetFileByContentHash(ctx, contentHash)
				existingFile = file
				return err
			})
		})

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
	storageKey := fmt.Sprintf("%s/%s/%s%s", input.UserID, time.Now().Format("2006/01/02"), contentHash, fileExt)

	storageURL, err := s.storageProvider.Upload(ctx, storageKey, &buf, input.ContentType)
	if err != nil {
		s.log.Error("Failed to upload to storage", logger.Error(err))
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	fileCategory := determineFileCategory(input.ContentType)

	var fileID string
	err = s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
			var deviceIDPtr *string
			if input.DeviceID != "" {
				deviceID := input.DeviceID
				deviceIDPtr = &deviceID
			}

			storageRegion := strings.TrimSpace(s.cfg.Storage.Region)
			var storageRegionPtr *string
			if storageRegion != "" {
				storageRegionPtr = &storageRegion
			}

			strPtr := func(s string) *string {
				if s == "" {
					return nil
				}
				return &s
			}

			cdnURL := s.buildCDNURL(storageKey)
			id, err := s.repo.CreateFile(ctx, dbModels.MediaFile{
				UploaderUserID:       input.UserID,
				FileName:             input.FileName,
				OriginalFileName:     strPtr(input.FileName),
				FileType:             input.ContentType,
				MimeType:             input.ContentType,
				FileCategory:         strPtr(fileCategory),
				FileExtension:        strPtr(strings.TrimPrefix(fileExt, ".")),
				FileSizeBytes:        input.FileSize,
				StorageProvider:      strPtr(s.cfg.Storage.Provider),
				StorageBucket:        strPtr(s.cfg.Storage.Bucket),
				StorageKey:           storageKey,
				StorageURL:           storageURL,
				StorageRegion:        storageRegionPtr,
				CDNURL:               strPtr(cdnURL),
				ContentHash:          strPtr(contentHash),
				Visibility:           dbModels.MediaVisibility(input.Visibility),
				UploadedFromDeviceID: deviceIDPtr,
				UploadedFromIP:       strPtr(input.IPAddress),
			})
			fileID = id
			return err
		})
	})

	if err != nil {
		s.log.Error("Failed to create file record", logger.Error(err))
		_ = s.storageProvider.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to create file record: %w", err)
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

func (s *MediaService) GetFile(ctx context.Context, input models.GetFileInput) (*models.GetFileOutput, error) {
	s.log.Info("Getting file",
		logger.String("file_id", input.FileID),
		logger.String("user_id", input.UserID),
	)

	var file *model.File
	err := s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
			fileParam, err := s.repo.GetFileByID(ctx, input.FileID)
			file = &model.File{
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
			return err
		})
	})

	if err != nil {
		s.log.Error("Failed to get file", logger.Error(err))
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if file == nil {
		s.log.Error("File not found")
		return nil, fmt.Errorf("file not found: %w", fmt.Errorf(mediaErrors.CodeFileNotFound))
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

	var file *model.File
	err := s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
			f, err := s.repo.GetFileByID(ctx, input.FileID)
			file = &model.File{
				ID:               f.ID,
				UploaderUserID:   f.UploaderUserID,
				FileName:         f.FileName,
				FileSizeBytes:    f.FileSizeBytes,
				FileType:         f.FileType,
				StorageURL:       f.StorageURL,
				CDNURL:           *f.CDNURL,
				ThumbnailURL:     *f.ThumbnailURL,
				ProcessingStatus: f.ProcessingStatus.String(),
				Visibility:       model.VisibilityType(f.Visibility),
				DownloadCount:    f.DownloadCount,
				ViewCount:        f.ViewCount,
				CreatedAt:        f.CreatedAt,
			}
			return err
		})
	})

	if err != nil || file == nil {
		return fmt.Errorf("file not found: %w", fmt.Errorf(mediaErrors.CodeFileNotFound))
	}

	if file.UploaderUserID != input.UserID {
		return fmt.Errorf("access denied: %w", fmt.Errorf(mediaErrors.CodeAccessDenied))
	}

	if input.Permanent {
		if err := s.storageProvider.Delete(ctx, file.StorageKey); err != nil {
			s.log.Error("Failed to delete from storage", logger.Error(err))
		}

		err = s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
			return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
				return s.repo.HardDeleteFile(ctx, input.FileID)
			})
		})
	} else {
		err = s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
			return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
				return s.repo.SoftDeleteFile(ctx, input.FileID)
			})
		})
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
