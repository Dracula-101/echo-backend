package storage

import (
	"context"
	"io"
	"time"
)

// Provider interface for object storage operations
type Provider interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)

	// UploadWithOptions uploads a file with custom options
	UploadWithOptions(ctx context.Context, key string, data io.Reader, opts UploadOptions) (string, error)

	// Download downloads a file from storage
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, key string) error

	// DeleteMultiple deletes multiple files from storage
	DeleteMultiple(ctx context.Context, keys []string) error

	// Exists checks if a file exists in storage
	Exists(ctx context.Context, key string) (bool, error)

	// GetMetadata retrieves metadata for a file
	GetMetadata(ctx context.Context, key string) (*Metadata, error)

	// GeneratePresignedURL generates a presigned URL for temporary access
	GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)

	// GenerateUploadURL generates a presigned URL for uploading
	GenerateUploadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)

	// ListObjects lists objects with a given prefix
	ListObjects(ctx context.Context, prefix string, maxKeys int) ([]ObjectInfo, error)
}

// UploadOptions contains options for uploading files
type UploadOptions struct {
	ContentType        string
	ContentDisposition string
	CacheControl       string
	ACL                string
	Metadata           map[string]string
	Tags               map[string]string
	ServerSideEncryption string
}

// Metadata represents file metadata
type Metadata struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
	Metadata     map[string]string
}

// ObjectInfo represents information about a stored object
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	StorageClass string
}

// Config contains common configuration for storage providers
type Config struct {
	Provider        string // r2, s3, local
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	PublicURL       string
	CDNBaseURL      string
	UseCDN          bool
	MaxFileSize     int64
}
