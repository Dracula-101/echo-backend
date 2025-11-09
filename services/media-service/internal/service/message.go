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

	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

func (s *MediaService) UploadMessageMedia(ctx context.Context, input models.UploadMessageMediaInput) (*models.UploadMessageMediaOutput, error) {
	s.log.Info("Uploading message media",
		logger.String("user_id", input.UserID),
		logger.String("conversation_id", input.ConversationID),
		logger.String("file_name", input.FileName),
	)

	exists, err := s.repo.ConversationExists(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate conversation: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("conversation not found")
	}

	if input.FileSize > s.cfg.Storage.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed")
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
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		ContentHash:    contentHash,
		FileExtension:  fileExt,
		Context:        dbModels.FileContextMessageMedia,
	})

	storageURL, err := s.storageProvider.Upload(ctx, storageKey, &buf, input.ContentType)
	if err != nil {
		s.log.Error("Failed to upload to storage", logger.Error(err))
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	fileCategory := determineFileCategory(input.ContentType)

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

			// Initialize JSON fields with empty values
			emptyArray := json.RawMessage("[]")
			emptyObject := json.RawMessage("{}")

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
				SubtitleTracks:       &emptyArray,
				ModerationLabels:     &emptyArray,
				ExifData:             &emptyObject,
				ProcessingStatus:     dbModels.FileProcessingStatusPending,
				ModerationStatus:     dbModels.ModerationStatusPending,
				Visibility:           dbModels.MediaVisibilityPrivate,
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
