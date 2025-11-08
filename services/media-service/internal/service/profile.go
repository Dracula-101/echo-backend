package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

func (s *MediaService) UploadProfilePhoto(ctx context.Context, input models.UploadProfilePhotoInput) (*models.UploadProfilePhotoOutput, error) {
	s.log.Info("Uploading profile photo",
		logger.String("user_id", input.UserID),
		logger.String("file_name", input.FileName),
	)

	maxProfilePhotoSize := int64(10 * 1024 * 1024)
	if input.FileSize > maxProfilePhotoSize {
		return nil, fmt.Errorf("profile photo too large: max %d MB", maxProfilePhotoSize/(1024*1024))
	}

	var buf bytes.Buffer
	teeReader := io.TeeReader(input.FileReader, &buf)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, teeReader); err != nil {
		s.log.Error("Failed to read file", logger.Error(err))
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	fileExt := filepath.Ext(input.FileName)
	storageKey := GenerateStoragePath(StoragePathConfig{
		UserID:        input.UserID,
		ContentHash:   contentHash,
		FileExtension: fileExt,
		Context:       dbModels.FileContextProfilePhoto,
	})

	storageURL, err := s.storageProvider.Upload(ctx, storageKey, &buf, input.ContentType)
	if err != nil {
		s.log.Error("Failed to upload to storage", logger.Error(err))
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	var fileID string
	err = s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
			strPtr := func(s string) *string {
				if s == "" {
					return nil
				}
				return &s
			}

			var deviceIDPtr *string
			if input.DeviceID != "" {
				deviceID := input.DeviceID
				deviceIDPtr = &deviceID
			}

			cdnURL := s.buildCDNURL(storageKey)
			fileCategory := "image"

			id, err := s.repo.CreateFile(ctx, dbModels.MediaFile{
				UploaderUserID:       input.UserID,
				FileName:             input.FileName,
				OriginalFileName:     strPtr(input.FileName),
				FileType:             input.ContentType,
				MimeType:             input.ContentType,
				FileCategory:         strPtr(fileCategory),
				FileExtension:        strPtr(filepath.Ext(input.FileName)[1:]),
				FileSizeBytes:        input.FileSize,
				StorageProvider:      strPtr(s.cfg.Storage.Provider),
				StorageBucket:        strPtr(s.cfg.Storage.Bucket),
				StorageKey:           storageKey,
				StorageURL:           storageURL,
				CDNURL:               strPtr(cdnURL),
				ContentHash:          strPtr(contentHash),
				Visibility:           dbModels.MediaVisibilityPublic,
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

	s.log.Info("Profile photo uploaded successfully", logger.String("file_id", fileID))

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
