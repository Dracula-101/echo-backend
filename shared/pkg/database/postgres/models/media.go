package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type MediaFile struct {
	ID             string `db:"id" json:"id" pk:"true"`
	UploaderUserID string `db:"uploader_user_id" json:"uploader_user_id"`

	// File details
	FileName         string  `db:"file_name" json:"file_name"`
	OriginalFileName *string `db:"original_file_name" json:"original_file_name,omitempty"`
	FileType         string  `db:"file_type" json:"file_type"`
	MimeType         string  `db:"mime_type" json:"mime_type"`
	FileCategory     *string `db:"file_category" json:"file_category,omitempty"`
	FileExtension    *string `db:"file_extension" json:"file_extension,omitempty"`
	FileSizeBytes    int64   `db:"file_size_bytes" json:"file_size_bytes"`

	// Storage details
	StorageProvider *string `db:"storage_provider" json:"storage_provider,omitempty"`
	StorageBucket   *string `db:"storage_bucket" json:"storage_bucket,omitempty"`
	StorageKey      string  `db:"storage_key" json:"storage_key"`
	StorageURL      string  `db:"storage_url" json:"storage_url"`
	StorageRegion   *string `db:"storage_region" json:"storage_region,omitempty"`
	CDNURL          *string `db:"cdn_url" json:"cdn_url,omitempty"`

	// File variants
	HasThumbnail       bool    `db:"has_thumbnail" json:"has_thumbnail"`
	ThumbnailURL       *string `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	ThumbnailSmallURL  *string `db:"thumbnail_small_url" json:"thumbnail_small_url,omitempty"`
	ThumbnailMediumURL *string `db:"thumbnail_medium_url" json:"thumbnail_medium_url,omitempty"`
	ThumbnailLargeURL  *string `db:"thumbnail_large_url" json:"thumbnail_large_url,omitempty"`
	HasPreview         bool    `db:"has_preview" json:"has_preview"`
	PreviewURL         *string `db:"preview_url" json:"preview_url,omitempty"`

	// Media-specific metadata
	Width           *int     `db:"width" json:"width,omitempty"`
	Height          *int     `db:"height" json:"height,omitempty"`
	DurationSeconds *int     `db:"duration_seconds" json:"duration_seconds,omitempty"`
	Bitrate         *int     `db:"bitrate" json:"bitrate,omitempty"`
	FrameRate       *float64 `db:"frame_rate" json:"frame_rate,omitempty"`
	Codec           *string  `db:"codec" json:"codec,omitempty"`
	Resolution      *string  `db:"resolution" json:"resolution,omitempty"`
	AspectRatio     *string  `db:"aspect_ratio" json:"aspect_ratio,omitempty"`
	ColorProfile    *string  `db:"color_profile" json:"color_profile,omitempty"`
	Orientation     *int     `db:"orientation" json:"orientation,omitempty"`

	// Image specific
	HasAlphaChannel *bool          `db:"has_alpha_channel" json:"has_alpha_channel,omitempty"`
	DominantColors  pq.StringArray `db:"dominant_colors" json:"dominant_colors,omitempty"`

	// Video specific
	VideoCodec     *string         `db:"video_codec" json:"video_codec,omitempty"`
	AudioCodec     *string         `db:"audio_codec" json:"audio_codec,omitempty"`
	SubtitleTracks json.RawMessage `db:"subtitle_tracks" json:"subtitle_tracks,omitempty"`

	// Audio specific
	AudioChannels *int `db:"audio_channels" json:"audio_channels,omitempty"`
	SampleRate    *int `db:"sample_rate" json:"sample_rate,omitempty"`

	// Document specific
	PageCount *int `db:"page_count" json:"page_count,omitempty"`
	WordCount *int `db:"word_count" json:"word_count,omitempty"`

	// Processing status
	ProcessingStatus      FileProcessingStatus `db:"processing_status" json:"processing_status"`
	ProcessingStartedAt   *time.Time           `db:"processing_started_at" json:"processing_started_at,omitempty"`
	ProcessingCompletedAt *time.Time           `db:"processing_completed_at" json:"processing_completed_at,omitempty"`
	ProcessingError       *string              `db:"processing_error" json:"processing_error,omitempty"`
	ProcessingAttempts    int                  `db:"processing_attempts" json:"processing_attempts"`

	// Security & Content
	IsEncrypted     bool             `db:"is_encrypted" json:"is_encrypted"`
	EncryptionKeyID *string          `db:"encryption_key_id" json:"encryption_key_id,omitempty"`
	ContentHash     *string          `db:"content_hash" json:"content_hash,omitempty"`
	Checksum        *string          `db:"checksum" json:"checksum,omitempty"`
	IsScanned       bool             `db:"is_scanned" json:"is_scanned"`
	VirusScanStatus *VirusScanStatus `db:"virus_scan_status" json:"virus_scan_status,omitempty"`
	VirusScanAt     *time.Time       `db:"virus_scan_at" json:"virus_scan_at,omitempty"`

	// Content moderation
	ModerationStatus  ModerationStatus `db:"moderation_status" json:"moderation_status"`
	ModerationScore   *float64         `db:"moderation_score" json:"moderation_score,omitempty"`
	ModerationLabels  json.RawMessage  `db:"moderation_labels" json:"moderation_labels,omitempty"`
	IsNSFW            bool             `db:"is_nsfw" json:"is_nsfw"`
	NSFWScore         *float64         `db:"nsfw_score" json:"nsfw_score,omitempty"`
	ModeratedAt       *time.Time       `db:"moderated_at" json:"moderated_at,omitempty"`
	ModeratedByUserID *string          `db:"moderated_by_user_id" json:"moderated_by_user_id,omitempty"`

	// Access control
	Visibility    MediaVisibility `db:"visibility" json:"visibility"`
	AccessToken   *string         `db:"access_token" json:"access_token,omitempty"`
	ExpiresAt     *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	MaxDownloads  *int            `db:"max_downloads" json:"max_downloads,omitempty"`
	DownloadCount int             `db:"download_count" json:"download_count"`
	ViewCount     int             `db:"view_count" json:"view_count"`

	// Compression
	IsCompressed          bool     `db:"is_compressed" json:"is_compressed"`
	CompressionRatio      *float64 `db:"compression_ratio" json:"compression_ratio,omitempty"`
	OriginalFileSizeBytes *int64   `db:"original_file_size_bytes" json:"original_file_size_bytes,omitempty"`

	// EXIF & Metadata
	ExifData     json.RawMessage `db:"exif_data" json:"exif_data,omitempty"`
	GPSLatitude  *float64        `db:"gps_latitude" json:"gps_latitude,omitempty"`
	GPSLongitude *float64        `db:"gps_longitude" json:"gps_longitude,omitempty"`
	GPSAltitude  *float64        `db:"gps_altitude" json:"gps_altitude,omitempty"`
	CameraMake   *string         `db:"camera_make" json:"camera_make,omitempty"`
	CameraModel  *string         `db:"camera_model" json:"camera_model,omitempty"`
	LensModel    *string         `db:"lens_model" json:"lens_model,omitempty"`
	FocalLength  *float64        `db:"focal_length" json:"focal_length,omitempty"`
	Aperture     *float64        `db:"aperture" json:"aperture,omitempty"`
	ISO          *int            `db:"iso" json:"iso,omitempty"`
	ShutterSpeed *string         `db:"shutter_speed" json:"shutter_speed,omitempty"`
	CaptureDate  *time.Time      `db:"capture_date" json:"capture_date,omitempty"`

	// Usage tracking
	LastAccessedAt *time.Time `db:"last_accessed_at" json:"last_accessed_at,omitempty"`
	AccessCount    int64      `db:"access_count" json:"access_count"`

	// Lifecycle
	UploadedAt          time.Time  `db:"uploaded_at" json:"uploaded_at"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt           *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
	PermanentlyDeleteAt *time.Time `db:"permanently_delete_at" json:"permanently_delete_at,omitempty"`

	// Device info
	UploadedFromDeviceID *string `db:"uploaded_from_device_id" json:"uploaded_from_device_id,omitempty"`
	UploadedFromIP       *string `db:"uploaded_from_ip" json:"uploaded_from_ip,omitempty"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (m *MediaFile) TableName() string {
	return "media.files"
}

func (m *MediaFile) PrimaryKey() interface{} {
	return m.ID
}

type ProcessingQueue struct {
	ID           string               `db:"id" json:"id" pk:"true"`
	FileID       string               `db:"file_id" json:"file_id"`
	TaskType     string               `db:"task_type" json:"task_type"`
	Priority     int                  `db:"priority" json:"priority"`
	Status       FileProcessingStatus `db:"status" json:"status"`
	AttemptCount int                  `db:"attempt_count" json:"attempt_count"`
	MaxAttempts  int                  `db:"max_attempts" json:"max_attempts"`
	WorkerID     *string              `db:"worker_id" json:"worker_id,omitempty"`
	StartedAt    *time.Time           `db:"started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time           `db:"completed_at" json:"completed_at,omitempty"`
	ErrorMessage *string              `db:"error_message" json:"error_message,omitempty"`
	InputParams  json.RawMessage      `db:"input_params" json:"input_params,omitempty"`
	OutputResult json.RawMessage      `db:"output_result" json:"output_result,omitempty"`
	CreatedAt    time.Time            `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time            `db:"updated_at" json:"updated_at"`
}

func (p *ProcessingQueue) TableName() string {
	return "media.processing_queue"
}

func (p *ProcessingQueue) PrimaryKey() interface{} {
	return p.ID
}

type Thumbnail struct {
	ID            string    `db:"id" json:"id" pk:"true"`
	FileID        string    `db:"file_id" json:"file_id"`
	SizeType      string    `db:"size_type" json:"size_type"`
	Width         int       `db:"width" json:"width"`
	Height        int       `db:"height" json:"height"`
	FileSizeBytes *int64    `db:"file_size_bytes" json:"file_size_bytes,omitempty"`
	StorageKey    string    `db:"storage_key" json:"storage_key"`
	StorageURL    string    `db:"storage_url" json:"storage_url"`
	Format        *string   `db:"format" json:"format,omitempty"`
	Quality       *int      `db:"quality" json:"quality,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

func (t *Thumbnail) TableName() string {
	return "media.thumbnails"
}

func (t *Thumbnail) PrimaryKey() interface{} {
	return t.ID
}

type TranscodingJob struct {
	ID                    string               `db:"id" json:"id" pk:"true"`
	SourceFileID          string               `db:"source_file_id" json:"source_file_id"`
	OutputFileID          *string              `db:"output_file_id" json:"output_file_id,omitempty"`
	ProfileName           string               `db:"profile_name" json:"profile_name"`
	Status                FileProcessingStatus `db:"status" json:"status"`
	ProgressPercentage    int                  `db:"progress_percentage" json:"progress_percentage"`
	StartedAt             *time.Time           `db:"started_at" json:"started_at,omitempty"`
	CompletedAt           *time.Time           `db:"completed_at" json:"completed_at,omitempty"`
	EstimatedCompletionAt *time.Time           `db:"estimated_completion_at" json:"estimated_completion_at,omitempty"`
	ErrorMessage          *string              `db:"error_message" json:"error_message,omitempty"`
	TranscodingParams     json.RawMessage      `db:"transcoding_params" json:"transcoding_params,omitempty"`
	CreatedAt             time.Time            `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time            `db:"updated_at" json:"updated_at"`
}

func (t *TranscodingJob) TableName() string {
	return "media.transcoding_jobs"
}

func (t *TranscodingJob) PrimaryKey() interface{} {
	return t.ID
}

type Album struct {
	ID            string          `db:"id" json:"id" pk:"true"`
	UserID        string          `db:"user_id" json:"user_id"`
	Title         string          `db:"title" json:"title"`
	Description   *string         `db:"description" json:"description,omitempty"`
	CoverFileID   *string         `db:"cover_file_id" json:"cover_file_id,omitempty"`
	AlbumType     AlbumType       `db:"album_type" json:"album_type"`
	IsSystemAlbum bool            `db:"is_system_album" json:"is_system_album"`
	FileCount     int             `db:"file_count" json:"file_count"`
	Visibility    MediaVisibility `db:"visibility" json:"visibility"`
	SortOrder     AlbumSortOrder  `db:"sort_order" json:"sort_order"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

func (a *Album) TableName() string {
	return "media.albums"
}

func (a *Album) PrimaryKey() interface{} {
	return a.ID
}

type AlbumFile struct {
	ID           string    `db:"id" json:"id" pk:"true"`
	AlbumID      string    `db:"album_id" json:"album_id"`
	FileID       string    `db:"file_id" json:"file_id"`
	DisplayOrder *int      `db:"display_order" json:"display_order,omitempty"`
	AddedAt      time.Time `db:"added_at" json:"added_at"`
}

func (a *AlbumFile) TableName() string {
	return "media.album_files"
}

func (a *AlbumFile) PrimaryKey() interface{} {
	return a.ID
}

type Tag struct {
	ID         string    `db:"id" json:"id" pk:"true"`
	UserID     *string   `db:"user_id" json:"user_id,omitempty"`
	TagName    string    `db:"tag_name" json:"tag_name"`
	TagType    TagType   `db:"tag_type" json:"tag_type"`
	UsageCount int       `db:"usage_count" json:"usage_count"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (t *Tag) TableName() string {
	return "media.tags"
}

func (t *Tag) PrimaryKey() interface{} {
	return t.ID
}

type FileTag struct {
	ID              string    `db:"id" json:"id" pk:"true"`
	FileID          string    `db:"file_id" json:"file_id"`
	TagID           string    `db:"tag_id" json:"tag_id"`
	ConfidenceScore *float64  `db:"confidence_score" json:"confidence_score,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

func (f *FileTag) TableName() string {
	return "media.file_tags"
}

func (f *FileTag) PrimaryKey() interface{} {
	return f.ID
}

type Share struct {
	ID                       string          `db:"id" json:"id" pk:"true"`
	FileID                   string          `db:"file_id" json:"file_id"`
	SharedByUserID           string          `db:"shared_by_user_id" json:"shared_by_user_id"`
	SharedWithUserID         *string         `db:"shared_with_user_id" json:"shared_with_user_id,omitempty"`
	SharedWithConversationID *string         `db:"shared_with_conversation_id" json:"shared_with_conversation_id,omitempty"`
	ShareToken               *string         `db:"share_token" json:"share_token,omitempty"`
	AccessType               ShareAccessType `db:"access_type" json:"access_type"`
	PasswordHash             *string         `db:"password_hash" json:"password_hash,omitempty"`
	ExpiresAt                *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	MaxViews                 *int            `db:"max_views" json:"max_views,omitempty"`
	ViewCount                int             `db:"view_count" json:"view_count"`
	DownloadCount            int             `db:"download_count" json:"download_count"`
	IsActive                 bool            `db:"is_active" json:"is_active"`
	CreatedAt                time.Time       `db:"created_at" json:"created_at"`
	RevokedAt                *time.Time      `db:"revoked_at" json:"revoked_at,omitempty"`
}

func (s *Share) TableName() string {
	return "media.shares"
}

func (s *Share) PrimaryKey() interface{} {
	return s.ID
}

type AccessLog struct {
	ID               string    `db:"id" json:"id" pk:"true"`
	FileID           string    `db:"file_id" json:"file_id"`
	UserID           *string   `db:"user_id" json:"user_id,omitempty"`
	AccessType       string    `db:"access_type" json:"access_type"`
	IPAddress        *string   `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent        *string   `db:"user_agent" json:"user_agent,omitempty"`
	DeviceID         *string   `db:"device_id" json:"device_id,omitempty"`
	Referrer         *string   `db:"referrer" json:"referrer,omitempty"`
	Success          bool      `db:"success" json:"success"`
	ErrorMessage     *string   `db:"error_message" json:"error_message,omitempty"`
	BytesTransferred *int64    `db:"bytes_transferred" json:"bytes_transferred,omitempty"`
	AccessDurationMS *int      `db:"access_duration_ms" json:"access_duration_ms,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

func (a *AccessLog) TableName() string {
	return "media.access_log"
}

func (a *AccessLog) PrimaryKey() interface{} {
	return a.ID
}

type StickerPack struct {
	ID              string    `db:"id" json:"id" pk:"true"`
	CreatorUserID   *string   `db:"creator_user_id" json:"creator_user_id,omitempty"`
	PackName        string    `db:"pack_name" json:"pack_name"`
	PackDescription *string   `db:"pack_description" json:"pack_description,omitempty"`
	CoverFileID     *string   `db:"cover_file_id" json:"cover_file_id,omitempty"`
	IconFileID      *string   `db:"icon_file_id" json:"icon_file_id,omitempty"`
	StickerCount    int       `db:"sticker_count" json:"sticker_count"`
	IsOfficial      bool      `db:"is_official" json:"is_official"`
	IsAnimated      bool      `db:"is_animated" json:"is_animated"`
	IsPublic        bool      `db:"is_public" json:"is_public"`
	DownloadCount   int       `db:"download_count" json:"download_count"`
	InstallCount    int       `db:"install_count" json:"install_count"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

func (s *StickerPack) TableName() string {
	return "media.sticker_packs"
}

func (s *StickerPack) PrimaryKey() interface{} {
	return s.ID
}

type Sticker struct {
	ID            string         `db:"id" json:"id" pk:"true"`
	CreatorUserID *string        `db:"creator_user_id" json:"creator_user_id,omitempty"`
	StickerPackID *string        `db:"sticker_pack_id" json:"sticker_pack_id,omitempty"`
	FileID        string         `db:"file_id" json:"file_id"`
	StickerName   *string        `db:"sticker_name" json:"sticker_name,omitempty"`
	Emojis        pq.StringArray `db:"emojis" json:"emojis,omitempty"`
	IsAnimated    bool           `db:"is_animated" json:"is_animated"`
	UsageCount    int64          `db:"usage_count" json:"usage_count"`
	IsActive      bool           `db:"is_active" json:"is_active"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
}

func (s *Sticker) TableName() string {
	return "media.stickers"
}

func (s *Sticker) PrimaryKey() interface{} {
	return s.ID
}

type UserStickerPack struct {
	ID            string    `db:"id" json:"id" pk:"true"`
	UserID        string    `db:"user_id" json:"user_id"`
	StickerPackID string    `db:"sticker_pack_id" json:"sticker_pack_id"`
	DisplayOrder  *int      `db:"display_order" json:"display_order,omitempty"`
	InstalledAt   time.Time `db:"installed_at" json:"installed_at"`
}

func (u *UserStickerPack) TableName() string {
	return "media.user_sticker_packs"
}

func (u *UserStickerPack) PrimaryKey() interface{} {
	return u.ID
}

type GIF struct {
	ID            string         `db:"id" json:"id" pk:"true"`
	Provider      *string        `db:"provider" json:"provider,omitempty"`
	ProviderGIFID *string        `db:"provider_gif_id" json:"provider_gif_id,omitempty"`
	Title         *string        `db:"title" json:"title,omitempty"`
	URL           string         `db:"url" json:"url"`
	PreviewURL    *string        `db:"preview_url" json:"preview_url,omitempty"`
	ThumbnailURL  *string        `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	Width         *int           `db:"width" json:"width,omitempty"`
	Height        *int           `db:"height" json:"height,omitempty"`
	FileSizeBytes *int64         `db:"file_size_bytes" json:"file_size_bytes,omitempty"`
	Tags          pq.StringArray `db:"tags" json:"tags,omitempty"`
	UsageCount    int64          `db:"usage_count" json:"usage_count"`
	IsTrending    bool           `db:"is_trending" json:"is_trending"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
}

func (g *GIF) TableName() string {
	return "media.gifs"
}

func (g *GIF) PrimaryKey() interface{} {
	return g.ID
}

type FavoriteGIF struct {
	ID      string    `db:"id" json:"id" pk:"true"`
	UserID  string    `db:"user_id" json:"user_id"`
	GIFID   string    `db:"gif_id" json:"gif_id"`
	AddedAt time.Time `db:"added_at" json:"added_at"`
}

func (f *FavoriteGIF) TableName() string {
	return "media.favorite_gifs"
}

func (f *FavoriteGIF) PrimaryKey() interface{} {
	return f.ID
}

type StorageStat struct {
	ID                    string    `db:"id" json:"id" pk:"true"`
	UserID                string    `db:"user_id" json:"user_id"`
	TotalFiles            int       `db:"total_files" json:"total_files"`
	TotalSizeBytes        int64     `db:"total_size_bytes" json:"total_size_bytes"`
	ImagesCount           int       `db:"images_count" json:"images_count"`
	ImagesSizeBytes       int64     `db:"images_size_bytes" json:"images_size_bytes"`
	VideosCount           int       `db:"videos_count" json:"videos_count"`
	VideosSizeBytes       int64     `db:"videos_size_bytes" json:"videos_size_bytes"`
	AudioCount            int       `db:"audio_count" json:"audio_count"`
	AudioSizeBytes        int64     `db:"audio_size_bytes" json:"audio_size_bytes"`
	DocumentsCount        int       `db:"documents_count" json:"documents_count"`
	DocumentsSizeBytes    int64     `db:"documents_size_bytes" json:"documents_size_bytes"`
	StorageQuotaBytes     int64     `db:"storage_quota_bytes" json:"storage_quota_bytes"`
	StorageUsedPercentage float64   `db:"storage_used_percentage" json:"storage_used_percentage"`
	LastCalculatedAt      time.Time `db:"last_calculated_at" json:"last_calculated_at"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
}

func (s *StorageStat) TableName() string {
	return "media.storage_stats"
}

func (s *StorageStat) PrimaryKey() interface{} {
	return s.ID
}
