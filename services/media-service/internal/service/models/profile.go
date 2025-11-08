package models

import (
	"io"
	"time"
)

// UploadProfilePhotoInput represents input for uploading a profile photo
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

// UploadProfilePhotoOutput represents output from uploading a profile photo
type UploadProfilePhotoOutput struct {
	FileID           string    `json:"file_id"`
	FileName         string    `json:"file_name"`
	FileSize         int64     `json:"file_size"`
	StorageURL       string    `json:"storage_url"`
	CDNURL           string    `json:"cdn_url,omitempty"`
	ThumbnailSmall   string    `json:"thumbnail_small,omitempty"`
	ThumbnailMedium  string    `json:"thumbnail_medium,omitempty"`
	ThumbnailLarge   string    `json:"thumbnail_large,omitempty"`
	ProcessingStatus string    `json:"processing_status"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

// UploadMessageMediaInput represents input for uploading message media
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

// UploadMessageMediaOutput represents output from uploading message media
type UploadMessageMediaOutput struct {
	FileID           string    `json:"file_id"`
	FileName         string    `json:"file_name"`
	FileSize         int64     `json:"file_size"`
	FileType         string    `json:"file_type"`
	StorageURL       string    `json:"storage_url"`
	CDNURL           string    `json:"cdn_url,omitempty"`
	ThumbnailURL     string    `json:"thumbnail_url,omitempty"`
	Width            int       `json:"width,omitempty"`
	Height           int       `json:"height,omitempty"`
	Duration         int       `json:"duration,omitempty"`
	ProcessingStatus string    `json:"processing_status"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

// GetStorageStatsInput represents input for getting storage statistics
type GetStorageStatsInput struct {
	UserID string
}

// GetStorageStatsOutput represents output from getting storage statistics
type GetStorageStatsOutput struct {
	UserID                string  `json:"user_id"`
	TotalFiles            int     `json:"total_files"`
	TotalSizeBytes        int64   `json:"total_size_bytes"`
	TotalSizeMB           float64 `json:"total_size_mb"`
	ImagesCount           int     `json:"images_count"`
	ImagesSizeBytes       int64   `json:"images_size_bytes"`
	VideosCount           int     `json:"videos_count"`
	VideosSizeBytes       int64   `json:"videos_size_bytes"`
	AudioCount            int     `json:"audio_count"`
	AudioSizeBytes        int64   `json:"audio_size_bytes"`
	DocumentsCount        int     `json:"documents_count"`
	DocumentsSizeBytes    int64   `json:"documents_size_bytes"`
	StorageQuotaBytes     int64   `json:"storage_quota_bytes"`
	StorageQuotaMB        float64 `json:"storage_quota_mb"`
	StorageUsedPercentage float64 `json:"storage_used_percentage"`
	LastCalculatedAt      string  `json:"last_calculated_at"`
}
