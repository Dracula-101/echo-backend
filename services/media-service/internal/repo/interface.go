package repo

import (
	"context"

	"media-service/internal/model"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
)

// FileRepositoryInterface defines the contract for file repository operations
type FileRepositoryInterface interface {
	// File operations
	CreateFile(ctx context.Context, model dbModels.MediaFile) (string, pkgErrors.AppError)
	GetFileByID(ctx context.Context, fileID string) (*dbModels.MediaFile, pkgErrors.AppError)
	ListFilesByUser(ctx context.Context, userID string, limit, offset int) ([]*dbModels.MediaFile, pkgErrors.AppError)
	UpdateFileProcessingStatus(ctx context.Context, fileID, status string, errorMsg string) pkgErrors.AppError
	UpdateImageMetadata(ctx context.Context, fileID string, width, height int, aspectRatio string, thumbnailSmallURL, thumbnailMediumURL, thumbnailLargeURL *string) pkgErrors.AppError
	IncrementDownloadCount(ctx context.Context, fileID string) pkgErrors.AppError
	IncrementViewCount(ctx context.Context, fileID string) pkgErrors.AppError
	SoftDeleteFile(ctx context.Context, fileID string) pkgErrors.AppError
	HardDeleteFile(ctx context.Context, fileID string) pkgErrors.AppError
	GetFileByContentHash(ctx context.Context, contentHash string) (*model.File, pkgErrors.AppError)

	// User storage operations
	CountFilesByUser(ctx context.Context, userID string) (int, pkgErrors.AppError)
	GetUserStorageUsage(ctx context.Context, userID string) (int64, pkgErrors.AppError)

	// Access logging
	CreateAccessLog(ctx context.Context, fileID, userID, accessType, ipAddress, userAgent, deviceID string, success bool, bytesTransferred int64) pkgErrors.AppError

	// Album operations
	CreateAlbum(ctx context.Context, album *dbModels.Album) (string, pkgErrors.AppError)
	GetAlbumByID(ctx context.Context, albumID string) (*dbModels.Album, pkgErrors.AppError)
	ListAlbumsByUser(ctx context.Context, userID string, limit, offset int) ([]*dbModels.Album, pkgErrors.AppError)
	UpdateAlbum(ctx context.Context, album *dbModels.Album) pkgErrors.AppError
	DeleteAlbum(ctx context.Context, albumID string) pkgErrors.AppError
	AddFileToAlbum(ctx context.Context, albumFile *dbModels.AlbumFile) pkgErrors.AppError
	RemoveFileFromAlbum(ctx context.Context, albumID, fileID string) pkgErrors.AppError
	ListAlbumFiles(ctx context.Context, albumID string, limit, offset int) ([]*dbModels.MediaFile, pkgErrors.AppError)

	// Share operations
	CreateShare(ctx context.Context, share *dbModels.Share) (string, pkgErrors.AppError)

	// Validation operations
	ConversationExists(ctx context.Context, conversationID string) (bool, pkgErrors.AppError)
	UserExists(ctx context.Context, userID string) (bool, pkgErrors.AppError)
	AlbumExistsAndOwned(ctx context.Context, albumID, userID string) (bool, pkgErrors.AppError)
	FileExistsAndOwned(ctx context.Context, fileID, userID string) (bool, pkgErrors.AppError)
}

// Ensure FileRepository implements FileRepositoryInterface
var _ FileRepositoryInterface = (*FileRepository)(nil)
