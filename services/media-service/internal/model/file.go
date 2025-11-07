package model

import (
	"time"
)

// File represents a media file in the system
type File struct {
	ID             string `json:"id"`
	UploaderUserID string `json:"uploader_user_id"`

	// File details
	FileName         string `json:"file_name"`
	OriginalFileName string `json:"original_file_name,omitempty"`
	FileType         string `json:"file_type"`
	MimeType         string `json:"mime_type"`
	FileCategory     string `json:"file_category,omitempty"`
	FileExtension    string `json:"file_extension,omitempty"`
	FileSizeBytes    int64  `json:"file_size_bytes"`

	// Storage details
	StorageProvider string `json:"storage_provider"`
	StorageBucket   string `json:"storage_bucket,omitempty"`
	StorageKey      string `json:"storage_key"`
	StorageURL      string `json:"storage_url"`
	StorageRegion   string `json:"storage_region,omitempty"`
	CDNURL          string `json:"cdn_url,omitempty"`

	// Thumbnails
	HasThumbnail       bool   `json:"has_thumbnail"`
	ThumbnailURL       string `json:"thumbnail_url,omitempty"`
	ThumbnailSmallURL  string `json:"thumbnail_small_url,omitempty"`
	ThumbnailMediumURL string `json:"thumbnail_medium_url,omitempty"`
	ThumbnailLargeURL  string `json:"thumbnail_large_url,omitempty"`
	HasPreview         bool   `json:"has_preview"`
	PreviewURL         string `json:"preview_url,omitempty"`

	// Media metadata
	Width           *int   `json:"width,omitempty"`
	Height          *int   `json:"height,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
	AspectRatio     string `json:"aspect_ratio,omitempty"`

	// Processing status
	ProcessingStatus string `json:"processing_status"`
	ProcessingError  string `json:"processing_error,omitempty"`

	// Security
	IsEncrypted     bool   `json:"is_encrypted"`
	ContentHash     string `json:"content_hash,omitempty"`
	IsScanned       bool   `json:"is_scanned"`
	VirusScanStatus string `json:"virus_scan_status,omitempty"`

	// Access control
	Visibility    VisibilityType `json:"visibility"`
	AccessToken   string         `json:"access_token,omitempty"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
	DownloadCount int            `json:"download_count"`
	ViewCount     int            `json:"view_count"`

	// Timestamps
	UploadedAt time.Time  `json:"uploaded_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type VisibilityType string

const (
	VisibilityPrivate VisibilityType = "private"
	VisibilityPublic  VisibilityType = "public"
)

func (v VisibilityType) String() string {
	return string(v)
}

// Album represents a collection of media files
type Album struct {
	ID          string    `json:"id"`
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

// Share represents a file share
type Share struct {
	ID               string     `json:"id"`
	FileID           string     `json:"file_id"`
	SharedByUserID   string     `json:"shared_by_user_id"`
	SharedWithUserID *string    `json:"shared_with_user_id,omitempty"`
	ShareToken       string     `json:"share_token,omitempty"`
	AccessType       string     `json:"access_type"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	MaxViews         *int       `json:"max_views,omitempty"`
	ViewCount        int        `json:"view_count"`
	DownloadCount    int        `json:"download_count"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
}

// StickerPack represents a sticker pack
type StickerPack struct {
	ID              string    `json:"id"`
	CreatorUserID   *string   `json:"creator_user_id,omitempty"`
	PackName        string    `json:"pack_name"`
	PackDescription string    `json:"pack_description,omitempty"`
	CoverFileID     *string   `json:"cover_file_id,omitempty"`
	StickerCount    int       `json:"sticker_count"`
	IsOfficial      bool      `json:"is_official"`
	IsAnimated      bool      `json:"is_animated"`
	IsPublic        bool      `json:"is_public"`
	DownloadCount   int       `json:"download_count"`
	InstallCount    int       `json:"install_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Sticker represents a sticker in a pack
type Sticker struct {
	ID            string    `json:"id"`
	CreatorUserID *string   `json:"creator_user_id,omitempty"`
	StickerPackID *string   `json:"sticker_pack_id,omitempty"`
	FileID        string    `json:"file_id"`
	StickerName   string    `json:"sticker_name,omitempty"`
	Emojis        []string  `json:"emojis,omitempty"`
	IsAnimated    bool      `json:"is_animated"`
	UsageCount    int64     `json:"usage_count"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}
