package domain

import "time"

// FileUploadResult captures the canonical response for a successful file upload.
type FileUploadResult struct {
	FileID           string
	FileName         string
	FileSize         int64
	FileType         string
	StorageURL       string
	CDNURL           string
	ProcessingStatus string
	AccessToken      string
	UploadedAt       time.Time
}

// MessageMediaResult represents the outcome of uploading media to a conversation.
type MessageMediaResult struct {
	FileID           string
	FileName         string
	FileSize         int64
	FileType         string
	StorageURL       string
	CDNURL           string
	ThumbnailURL     string
	Width            int
	Height           int
	Duration         int
	ProcessingStatus string
	UploadedAt       time.Time
}

// ProfilePhotoResult contains processed profile photo metadata.
type ProfilePhotoResult struct {
	FileID           string
	FileName         string
	FileSize         int64
	StorageURL       string
	CDNURL           string
	ThumbnailSmall   string
	ThumbnailMedium  string
	ThumbnailLarge   string
	ProcessingStatus string
	UploadedAt       time.Time
}

// FileMetadataResult describes a stored file returned via lookup.
type FileMetadataResult struct {
	FileID           string
	FileName         string
	FileSize         int64
	FileType         string
	StorageURL       string
	CDNURL           string
	ThumbnailURL     string
	ProcessingStatus string
	Visibility       string
	DownloadCount    int
	ViewCount        int
}

// DeleteFileResult communicates the outcome of a delete operation.
type DeleteFileResult struct {
	FileID  string
	Message string
}

// AlbumSummary provides a lightweight view of a media album.
type AlbumSummary struct {
	AlbumID     string
	UserID      string
	Title       string
	Description string
	CoverFileID string
	AlbumType   string
	FileCount   int
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AlbumListResult bundles paginated album metadata.
type AlbumListResult struct {
	Albums []AlbumSummary
	Limit  int
	Offset int
}

// AlbumMutationResult is returned when files are added to or removed from an album.
type AlbumMutationResult struct {
	AlbumID string
	FileID  string
	Message string
}

// ShareCreationResult captures the identifiers tied to a share link.
type ShareCreationResult struct {
	ShareID    string
	ShareToken string
	ShareURL   string
	ExpiresAt  *time.Time
}

// ShareRevocationResult indicates that a share link has been revoked.
type ShareRevocationResult struct {
	ShareID string
	Message string
}

// StorageStatsResult reports a user's storage consumption breakdown.
type StorageStatsResult struct {
	UserID                string
	TotalFiles            int
	TotalSizeBytes        int64
	TotalSizeMB           float64
	ImagesCount           int
	ImagesSizeBytes       int64
	VideosCount           int
	VideosSizeBytes       int64
	AudioCount            int
	AudioSizeBytes        int64
	DocumentsCount        int
	DocumentsSizeBytes    int64
	StorageQuotaBytes     int64
	StorageQuotaMB        float64
	StorageUsedPercentage float64
	LastCalculatedAt      string
}
