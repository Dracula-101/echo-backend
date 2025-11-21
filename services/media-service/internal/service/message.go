package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"media-service/internal/errors"
	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

func (s *MediaService) UploadMessageMedia(ctx context.Context, input models.UploadMessageMediaInput) (*models.UploadMessageMediaOutput, pkgErrors.AppError) {
	s.log.Info("Uploading message media",
		logger.String("user_id", input.UserID),
		logger.String("conversation_id", input.ConversationID),
		logger.String("file_name", input.FileName),
	)

	exists, err := s.repo.ConversationExists(ctx, input.ConversationID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to verify conversation existence").
			WithDetail("conversation_id", input.ConversationID).
			WithService("media_service")
	}
	if !exists {
		return nil, pkgErrors.New(errors.CodeConversationNotFound, "conversation does not exist").
			WithDetail("conversation_id", input.ConversationID).
			WithService("media_service")
	}

	if input.FileSize > s.cfg.Storage.MaxFileSize {
		return nil, pkgErrors.New(errors.CodeFileTooLarge, "file size exceeds maximum limit").
			WithDetail("file_size", fmt.Sprintf("%d", input.FileSize)).
			WithDetail("max_file_size", fmt.Sprintf("%d", s.cfg.Storage.MaxFileSize)).
			WithService("media_service")
	}

	var buf bytes.Buffer
	teeReader := io.TeeReader(input.FileReader, &buf)

	hasher := sha256.New()
	if _, err := io.Copy(hasher, teeReader); err != nil {
		s.log.Error("Failed to read file", logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to read file").
			WithService("media_service")
	}
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	fileExt := filepath.Ext(input.FileName)
	storageKey := GenerateStoragePath(StoragePathConfig{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		ContentHash:    contentHash,
		FileExtension:  fileExt,
		Context:        dbModels.FileContextMessageMedia,
	})

	storageURL, storageErr := s.storageProvider.Upload(ctx, storageKey, &buf, input.ContentType)
	if storageErr != nil {
		s.log.Error("Failed to upload to storage", logger.Error(storageErr))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to upload file").
			WithService("media_service")
	}

	fileCategory := determineFileCategory(input.ContentType)

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

	// Initialize JSON fields with empty values
	emptyArray := json.RawMessage("[]")
	emptyObject := json.RawMessage("{}")

	fileID, err := s.repo.CreateFile(ctx, dbModels.MediaFile{
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
		SubtitleTracks:       &emptyArray,
		ModerationLabels:     &emptyArray,
		ExifData:             &emptyObject,
		ProcessingStatus:     dbModels.FileProcessingStatusPending,
		ModerationStatus:     dbModels.ModerationStatusPending,
		Visibility:           dbModels.MediaVisibilityPrivate,
		UploadedFromDeviceID: deviceIDPtr,
		UploadedFromIP:       strPtr(input.IPAddress),
	})

	if err != nil {
		s.log.Error("Failed to create file record", logger.Error(err))
		_ = s.storageProvider.Delete(ctx, storageKey)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create file record").WithDetail("user_id", input.UserID).
			WithDetail("conversation_id", input.ConversationID).
			WithService("media_service")
	}

	s.log.Info("Message media uploaded successfully", logger.String("file_id", fileID))

	return &models.UploadMessageMediaOutput{
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
