package service

import (
	"context"
	"fmt"
	pkgErrors "shared/pkg/errors"
	"time"

	"media-service/internal/service/models"
)

func (s *MediaService) GetStorageStats(ctx context.Context, input models.GetStorageStatsInput) (*models.GetStorageStatsOutput, pkgErrors.AppError) {
	stats, err := s.repo.GetStorageStats(ctx, input.UserID)

	if err != nil || stats == nil || time.Since(stats.LastCalculatedAt) > time.Hour {
		stats, err = s.repo.CalculateStorageStats(ctx, input.UserID)
		if err != nil {
			return nil, pkgErrors.FromError(fmt.Errorf("failed to calculate storage stats: %w", err), pkgErrors.CodeInternal, "failed to calculate storage stats").
				WithDetail("user_id", input.UserID).
				WithService("media-service")	
		}

		_ = s.repo.CreateOrUpdateStorageStats(ctx, stats)
	}

	return &models.GetStorageStatsOutput{
		UserID:                input.UserID,
		TotalFiles:            stats.TotalFiles,
		TotalSizeBytes:        stats.TotalSizeBytes,
		TotalSizeMB:           float64(stats.TotalSizeBytes) / (1024 * 1024),
		ImagesCount:           stats.ImagesCount,
		ImagesSizeBytes:       stats.ImagesSizeBytes,
		VideosCount:           stats.VideosCount,
		VideosSizeBytes:       stats.VideosSizeBytes,
		AudioCount:            stats.AudioCount,
		AudioSizeBytes:        stats.AudioSizeBytes,
		DocumentsCount:        stats.DocumentsCount,
		DocumentsSizeBytes:    stats.DocumentsSizeBytes,
		StorageQuotaBytes:     stats.StorageQuotaBytes,
		StorageQuotaMB:        float64(stats.StorageQuotaBytes) / (1024 * 1024),
		StorageUsedPercentage: stats.StorageUsedPercentage,
		LastCalculatedAt:      stats.LastCalculatedAt.Format(time.RFC3339),
	}, nil
}
