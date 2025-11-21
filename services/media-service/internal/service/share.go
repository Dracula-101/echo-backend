package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
)

func (s *MediaService) CreateShare(ctx context.Context, input models.CreateShareInput) (*models.CreateShareOutput, pkgErrors.AppError) {
	file, repoErr := s.repo.GetFileByID(ctx, input.FileID)
	if repoErr != nil || file == nil {
		return nil, pkgErrors.FromError(fmt.Errorf("file not found"), pkgErrors.CodeNotFound, "file not found").
			WithService("media-service")
	}

	if file.UploaderUserID != input.UserID {
		return nil, pkgErrors.FromError(fmt.Errorf("access denied"), pkgErrors.CodePermissionDenied, "access denied").
			WithService("media-service")
	}

	token, tokenErr := generateShareToken()
	if tokenErr != nil {
		return nil, pkgErrors.FromError(fmt.Errorf("failed to generate share token: %w", tokenErr), pkgErrors.CodeInternal, "failed to generate share token").
			WithService("media-service")
	}

	var expiresAt *time.Time
	if input.ExpiresIn != nil {
		expires := time.Now().Add(*input.ExpiresIn)
		expiresAt = &expires
	}

	var sharedWithUserID *string
	if input.SharedWithUser != "" {
		sharedWithUserID = &input.SharedWithUser
	}

	var conversationID *string
	if input.ConversationID != "" {
		conversationID = &input.ConversationID
	}

	share := &dbModels.Share{
		FileID:                   input.FileID,
		SharedByUserID:           input.UserID,
		SharedWithUserID:         sharedWithUserID,
		SharedWithConversationID: conversationID,
		ShareToken:               &token,
		AccessType:               dbModels.ShareAccessType(input.AccessType),
		ExpiresAt:                expiresAt,
		MaxViews:                 input.MaxViews,
		IsActive:                 true,
	}

	shareID, execErr := s.repo.CreateShare(ctx, share)
	if execErr != nil {
		return nil, pkgErrors.FromError(execErr, pkgErrors.CodeInternal, "failed to create share").
			WithDetail("file_id", input.FileID).
			WithDetail("user_id", input.UserID).
			WithService("media-service")
	}

	shareURL := fmt.Sprintf("%s/share/%s", s.cfg.Server.Host, token)

	return &models.CreateShareOutput{
		ShareID:    shareID,
		ShareToken: token,
		ShareURL:   shareURL,
		ExpiresAt:  expiresAt,
	}, nil
}

func (s *MediaService) RevokeShare(ctx context.Context, input models.RevokeShareInput) pkgErrors.AppError {
	share, err := s.repo.GetShareByID(ctx, input.ShareID)
	if err != nil || share == nil {
		return pkgErrors.New(pkgErrors.CodeNotFound, "share not found").
			WithDetail("share_id", input.ShareID).
			WithService("media-service")
	}

	if share.SharedByUserID != input.UserID {
		return pkgErrors.New(pkgErrors.CodePermissionDenied, "access denied").
			WithDetail("share_id", input.ShareID).
			WithDetail("user_id", input.UserID).
			WithService("media-service")
	}

	return s.repo.RevokeShare(ctx, input.ShareID)
}

func generateShareToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
