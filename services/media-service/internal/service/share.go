package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
)

func (s *MediaService) CreateShare(ctx context.Context, input models.CreateShareInput) (*models.CreateShareOutput, error) {
	file, err := s.repo.GetFileByID(ctx, input.FileID)
	if err != nil || file == nil {
		return nil, fmt.Errorf("file not found")
	}

	if file.UploaderUserID != input.UserID {
		return nil, fmt.Errorf("access denied")
	}

	token, err := generateShareToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate share token: %w", err)
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

	var shareID string
	err = s.dbCircuit.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return s.retryer.DoWithContext(ctx, func(ctx context.Context) error {
			id, err := s.repo.CreateShare(ctx, share)
			shareID = id
			return err
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create share: %w", err)
	}

	shareURL := fmt.Sprintf("%s/share/%s", s.cfg.Server.Host, token)

	return &models.CreateShareOutput{
		ShareID:    shareID,
		ShareToken: token,
		ShareURL:   shareURL,
		ExpiresAt:  expiresAt,
		CreatedAt:  time.Now(),
	}, nil
}

func (s *MediaService) RevokeShare(ctx context.Context, input models.RevokeShareInput) error {
	share, err := s.repo.GetShareByID(ctx, input.ShareID)
	if err != nil || share == nil {
		return fmt.Errorf("share not found")
	}

	if share.SharedByUserID != input.UserID {
		return fmt.Errorf("access denied")
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
