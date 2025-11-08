package models

import (
	"time"
)

// GetFileInput represents input for getting a file
type GetFileInput struct {
	FileID         string
	UserID         string
	AccessToken    string
	AllowedFormats []string
}

// GetFileOutput represents output from getting a file
type GetFileOutput struct {
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
	CreatedAt        time.Time
}

// ListFilesInput represents input for listing files
type ListFilesInput struct {
	UserID       string
	FileCategory string
	Limit        int
	Offset       int
	SortBy       string
	SortOrder    string
}

// ListFilesOutput represents output from listing files
type ListFilesOutput struct {
	Files      []FileItem
	TotalCount int
	HasMore    bool
}

// FileItem represents a file in a list
type FileItem struct {
	FileID       string
	FileName     string
	FileSize     int64
	FileType     string
	ThumbnailURL string
	CreatedAt    time.Time
}

// DeleteFileInput represents input for deleting a file
type DeleteFileInput struct {
	FileID    string
	UserID    string
	Permanent bool
}

// CreateAlbumInput represents input for creating an album
type CreateAlbumInput struct {
	UserID      string
	Title       string
	Description string
	AlbumType   string
	Visibility  string
}

// CreateAlbumOutput represents output from creating an album
type CreateAlbumOutput struct {
	AlbumID   string
	Title     string
	AlbumType string
	CreatedAt time.Time
}

// AddFileToAlbumInput represents input for adding a file to an album
type AddFileToAlbumInput struct {
	AlbumID      string
	FileID       string
	UserID       string
	DisplayOrder int
}

// GetAlbumInput represents input for getting an album
type GetAlbumInput struct {
	AlbumID string
	UserID  string
}

// GetAlbumOutput represents output from getting an album
type GetAlbumOutput struct {
	AlbumID     string    `json:"album_id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	CoverFileID string    `json:"cover_file_id,omitempty"`
	AlbumType   string    `json:"album_type"`
	FileCount   int       `json:"file_count"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListAlbumsInput represents input for listing albums
type ListAlbumsInput struct {
	UserID string
	Limit  int
	Offset int
}

// RemoveFileFromAlbumInput represents input for removing a file from an album
type RemoveFileFromAlbumInput struct {
	AlbumID string
	FileID  string
	UserID  string
}

// CreateShareInput represents input for creating a share
type CreateShareInput struct {
	FileID         string
	UserID         string
	SharedWithUser string
	ConversationID string
	AccessType     string
	ExpiresIn      *time.Duration
	MaxViews       *int
	Password       string
}

// CreateShareOutput represents output from creating a share
type CreateShareOutput struct {
	ShareID    string
	ShareToken string
	ShareURL   string
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

// GetSharedFileInput represents input for getting a shared file
type GetSharedFileInput struct {
	ShareToken string
	Password   string
	UserID     string
}

// RevokeShareInput represents input for revoking a share
type RevokeShareInput struct {
	ShareID string
	UserID  string
}
