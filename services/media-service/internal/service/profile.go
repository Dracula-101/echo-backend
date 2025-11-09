package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"
	"time"

	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

func (s *MediaService) UploadProfilePhoto(ctx context.Context, input models.UploadProfilePhotoInput) (*models.UploadProfilePhotoOutput, pkgErrors.AppError) {
	maxProfilePhotoSize := int64(10 * 1024 * 1024)
	if input.FileSize > maxProfilePhotoSize {
		return nil, pkgErrors.New(pkgErrors.CodeValidationFailed, "profile photo too large").
			WithDetail("max_size_mb", maxProfilePhotoSize/(1024*1024)).
			WithDetail("file_size", input.FileSize).
			WithService("media-service")
	}
	s.log.Debug("Uploading profile photo",
		logger.String("user_id", input.UserID),
		logger.String("file_name", input.FileName),
		logger.Int64("file_size", input.FileSize),
		logger.String("content_type", input.ContentType),
	)

	var buf bytes.Buffer
	teeReader := io.TeeReader(input.FileReader, &buf)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, teeReader); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to read file").
			WithDetail("file_name", input.FileName).
			WithService("media-service")
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	fileExt := filepath.Ext(input.FileName)
	storageKey := GenerateStoragePath(StoragePathConfig{
		UserID:        input.UserID,
		ContentHash:   contentHash,
		FileExtension: fileExt,
		Context:       dbModels.FileContextProfilePhoto,
	})

	storageURL, storageErr := s.storageProvider.Upload(ctx, storageKey, &buf, input.ContentType)
	if storageErr != nil {
		return nil, pkgErrors.FromError(storageErr, pkgErrors.CodeInternal, "failed to upload file to storage").
			WithDetail("storage_key", storageKey).
			WithDetail("content_type", input.ContentType).
			WithService("media-service")
	}
	s.log.Debug("File uploaded to storage",
		logger.String("storage_key", storageKey),
		logger.String("storage_url", storageURL),
	)

	cdnURL := s.buildCDNURL(storageKey)
	fileID, dbErr := s.repo.CreateFile(ctx, dbModels.MediaFile{
		UploaderUserID:       input.UserID,
		FileName:             input.FileName,
		OriginalFileName:     utils.PtrString(input.FileName),
		FileType:             input.ContentType,
		MimeType:             input.ContentType,
		FileCategory:         utils.PtrString("image"),
		FileExtension:        utils.PtrString(filepath.Ext(input.FileName)[1:]),
		FileSizeBytes:        input.FileSize,
		StorageProvider:      utils.PtrString(s.cfg.Storage.Provider),
		StorageBucket:        utils.PtrString(s.cfg.Storage.Bucket),
		StorageKey:           storageKey,
		StorageURL:           storageURL,
		CDNURL:               utils.PtrString(cdnURL),
		ContentHash:          utils.PtrString(contentHash),
		ProcessingStatus:     dbModels.FileProcessingStatusPending,
		ModerationStatus:     dbModels.ModerationStatusPending,
		Visibility:           dbModels.MediaVisibilityPublic,
		UploadedFromDeviceID: utils.PtrString(input.DeviceID),
		UploadedFromIP:       utils.PtrString(input.IPAddress),
	})

	if dbErr != nil {
		s.storageProvider.Delete(ctx, storageKey)
		return nil, pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to create file record in database")
	}

	return &models.UploadProfilePhotoOutput{
		FileID:           fileID,
		FileName:         input.FileName,
		FileSize:         input.FileSize,
		StorageURL:       storageURL,
		CDNURL:           s.buildCDNURL(storageKey),
		ProcessingStatus: "pending",
		UploadedAt:       time.Now(),
	}, nil
}
