package domain

import (
	"io"
	"time"
)

// UploadFileInput captures the information required to ingest a generic media file.
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
	Metadata    map[string]any
}

// UploadProfilePhotoInput encapsulates the data needed to upload or replace a profile avatar.
type UploadProfilePhotoInput struct {
	UserID      string
	FileReader  io.Reader
	FileName    string
	FileSize    int64
	ContentType string
	DeviceID    string
	IPAddress   string
	UserAgent   string
}

// UploadMessageMediaInput contains the payload for uploading conversation-scoped media.
type UploadMessageMediaInput struct {
	UserID         string
	ConversationID string
	MessageID      string
	FileReader     io.Reader
	FileName       string
	FileSize       int64
	ContentType    string
	Caption        string
	DeviceID       string
	IPAddress      string
	UserAgent      string
}

// GetFileInput describes how a stored asset should be looked up.
type GetFileInput struct {
	FileID         string
	UserID         string
	AccessToken    string
	AllowedFormats []string
}

// DeleteFileInput conveys the user's intent to delete or purge a file.
type DeleteFileInput struct {
	FileID    string
	UserID    string
	Permanent bool
}

// CreateAlbumInput defines the attributes needed to create a media album.
type CreateAlbumInput struct {
	UserID      string
	Title       string
	Description string
	AlbumType   string
	Visibility  string
}

// GetAlbumInput identifies a specific album for retrieval.
type GetAlbumInput struct {
	AlbumID string
	UserID  string
}

// ListAlbumsInput specifies pagination filters for fetching a user's albums.
type ListAlbumsInput struct {
	UserID string
	Limit  int
	Offset int
}

// AddFileToAlbumInput links an uploaded asset to a given album.
type AddFileToAlbumInput struct {
	AlbumID      string
	FileID       string
	UserID       string
	DisplayOrder int
}

// RemoveFileFromAlbumInput detaches an asset from an album context.
type RemoveFileFromAlbumInput struct {
	AlbumID string
	FileID  string
	UserID  string
}

// CreateShareInput captures all metadata necessary to mint a share link.
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

// RevokeShareInput represents a request to invalidate an existing share link.
type RevokeShareInput struct {
	ShareID string
	UserID  string
}

// GetStorageStatsInput scopes a request for storage usage insights.
type GetStorageStatsInput struct {
	UserID string
}
