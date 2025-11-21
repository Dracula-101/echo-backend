package service

import (
	"context"

	"media-service/internal/service/models"

	pkgErrors "shared/pkg/errors"
)

// MediaServiceInterface defines the contract for media service operations
type MediaServiceInterface interface {
	// File operations
	UploadFile(ctx context.Context, input models.UploadFileInput) (*models.UploadFileOutput, pkgErrors.AppError)
	GetFile(ctx context.Context, input models.GetFileInput) (*models.GetFileOutput, pkgErrors.AppError)
	DeleteFile(ctx context.Context, input models.DeleteFileInput) pkgErrors.AppError

	// Album operations
	CreateAlbum(ctx context.Context, input models.CreateAlbumInput) (*models.CreateAlbumOutput, pkgErrors.AppError)
	GetAlbum(ctx context.Context, input models.GetAlbumInput) (*models.GetAlbumOutput, pkgErrors.AppError)
	ListAlbums(ctx context.Context, input models.ListAlbumsInput) ([]*models.GetAlbumOutput, pkgErrors.AppError)
	AddFileToAlbum(ctx context.Context, input models.AddFileToAlbumInput) pkgErrors.AppError
	RemoveFileFromAlbum(ctx context.Context, input models.RemoveFileFromAlbumInput) pkgErrors.AppError

	// Message media operations
	UploadMessageMedia(ctx context.Context, input models.UploadMessageMediaInput) (*models.UploadMessageMediaOutput, pkgErrors.AppError)

	// Profile operations
	UploadProfilePhoto(ctx context.Context, input models.UploadProfilePhotoInput) (*models.UploadProfilePhotoOutput, pkgErrors.AppError)

	// Share operations
	CreateShare(ctx context.Context, input models.CreateShareInput) (*models.CreateShareOutput, pkgErrors.AppError)
	RevokeShare(ctx context.Context, input models.RevokeShareInput) pkgErrors.AppError

	// Stats operations
	GetStorageStats(ctx context.Context, input models.GetStorageStatsInput) (*models.GetStorageStatsOutput, pkgErrors.AppError)
}

// Ensure MediaService implements MediaServiceInterface
var _ MediaServiceInterface = (*MediaService)(nil)
