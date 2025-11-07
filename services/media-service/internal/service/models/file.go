package models

import (
	"io"
	"time"
)

// UploadFileInput represents input for uploading a file
type UploadFileInput struct {
	UserID      string
	FileReader  io.Reader
	FileName    string
	FileSize    int64
	ContentType string
	Visibility  string
	DeviceID    string
	IPAddress   string
	UserAgent   string
	Metadata    map[string]interface{}
}

// UploadFileOutput represents output from file upload
type UploadFileOutput struct {
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

// CreateShareInput represents input for creating a share
type CreateShareInput struct {
	FileID     string
	UserID     string
	AccessType string
	ExpiresIn  *time.Duration
	MaxViews   *int
	Password   string
}

// CreateShareOutput represents output from creating a share
type CreateShareOutput struct {
	ShareID    string
	ShareToken string
	ShareURL   string
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}
