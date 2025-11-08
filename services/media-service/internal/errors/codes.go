package errors

const (
	ServiceName = "media-service"
)

// Media service specific error codes
const (
	// Upload errors
	CodeUploadFailed         = "MEDIA_UPLOAD_FAILED"
	CodeFileTooLarge         = "MEDIA_FILE_TOO_LARGE"
	CodeInvalidFileType      = "MEDIA_INVALID_FILE_TYPE"
	CodeInvalidFileName      = "MEDIA_INVALID_FILE_NAME"
	CodeUploadTimeout        = "MEDIA_UPLOAD_TIMEOUT"
	CodeStorageQuotaExceeded = "MEDIA_STORAGE_QUOTA_EXCEEDED"

	// Download errors
	CodeDownloadFailed     = "MEDIA_DOWNLOAD_FAILED"
	CodeFileNotFound       = "MEDIA_FILE_NOT_FOUND"
	CodeFileExpired        = "MEDIA_FILE_EXPIRED"
	CodeAccessDenied       = "MEDIA_ACCESS_DENIED"
	CodeInvalidAccessToken = "MEDIA_INVALID_ACCESS_TOKEN"

	// Processing errors
	CodeProcessingFailed  = "MEDIA_PROCESSING_FAILED"
	CodeThumbnailFailed   = "MEDIA_THUMBNAIL_GENERATION_FAILED"
	CodeTranscodingFailed = "MEDIA_TRANSCODING_FAILED"
	CodeCompressionFailed = "MEDIA_COMPRESSION_FAILED"

	// Security errors
	CodeVirusScanFailed  = "MEDIA_VIRUS_SCAN_FAILED"
	CodeVirusDetected    = "MEDIA_VIRUS_DETECTED"
	CodeModerationFailed = "MEDIA_MODERATION_FAILED"
	CodeContentRejected  = "MEDIA_CONTENT_REJECTED"
	CodeNSFWDetected     = "MEDIA_NSFW_DETECTED"

	// Album errors
	CodeAlbumNotFound       = "MEDIA_ALBUM_NOT_FOUND"
	CodeAlbumCreationFailed = "MEDIA_ALBUM_CREATION_FAILED"
	CodeAlbumLimitExceeded  = "MEDIA_ALBUM_LIMIT_EXCEEDED"
	CodeFileAlreadyInAlbum  = "MEDIA_FILE_ALREADY_IN_ALBUM"

	// Sticker errors
	CodeStickerPackNotFound  = "MEDIA_STICKER_PACK_NOT_FOUND"
	CodeStickerNotFound      = "MEDIA_STICKER_NOT_FOUND"
	CodeStickerLimitExceeded = "MEDIA_STICKER_LIMIT_EXCEEDED"

	// Share errors
	CodeShareCreationFailed = "MEDIA_SHARE_CREATION_FAILED"
	CodeShareNotFound       = "MEDIA_SHARE_NOT_FOUND"
	CodeShareExpired        = "MEDIA_SHARE_EXPIRED"
	CodeShareLimitExceeded  = "MEDIA_SHARE_LIMIT_EXCEEDED"

	// Validation errors
	CodeInvalidMetadata   = "MEDIA_INVALID_METADATA"
	CodeInvalidDimensions = "MEDIA_INVALID_DIMENSIONS"
	CodeInvalidDuration   = "MEDIA_INVALID_DURATION"
	CodeInvalidFormat     = "MEDIA_INVALID_FORMAT"
	CodeCorruptedFile     = "MEDIA_CORRUPTED_FILE"

	// General errors
	CodeInternalError      = "MEDIA_INTERNAL_ERROR"
	CodeServiceUnavailable = "MEDIA_SERVICE_UNAVAILABLE"
)
