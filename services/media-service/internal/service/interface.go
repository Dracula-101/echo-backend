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
	GetFile(ctx context.Context, input models.GetFileInput) (*models.GetFileOutput, error)
	DeleteFile(ctx context.Context, input models.DeleteFileInput) error

	// Album operations
	CreateAlbum(ctx context.Context, input models.CreateAlbumInput) (*models.CreateAlbumOutput, error)
	GetAlbum(ctx context.Context, input models.GetAlbumInput) (*models.GetAlbumOutput, error)
	ListAlbums(ctx context.Context, input models.ListAlbumsInput) ([]*models.GetAlbumOutput, error)
	AddFileToAlbum(ctx context.Context, input models.AddFileToAlbumInput) error
	RemoveFileFromAlbum(ctx context.Context, input models.RemoveFileFromAlbumInput) error

	// Message media operations
	UploadMessageMedia(ctx context.Context, input models.UploadMessageMediaInput) (*models.UploadMessageMediaOutput, error)

	// Profile operations
	UploadProfilePhoto(ctx context.Context, input models.UploadProfilePhotoInput) (*models.UploadProfilePhotoOutput, pkgErrors.AppError)

	// Share operations
	CreateShare(ctx context.Context, input models.CreateShareInput) (*models.CreateShareOutput, pkgErrors.AppError)
	RevokeShare(ctx context.Context, input models.RevokeShareInput) error

	// Stats operations
	GetStorageStats(ctx context.Context, input models.GetStorageStatsInput) (*models.GetStorageStatsOutput, error)
}

// Ensure MediaService implements MediaServiceInterface
var _ MediaServiceInterface = (*MediaService)(nil)
