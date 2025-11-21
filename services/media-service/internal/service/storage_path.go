package service

import (
	"fmt"
	"path/filepath"
	"time"

	"shared/pkg/database/postgres/models"
)

// StoragePathConfig contains configuration for generating storage paths
type StoragePathConfig struct {
	UserID         string
	ConversationID string
	AlbumID        string
	StickerPackID  string
	ContentHash    string
	FileExtension  string
	Context        models.FileContext
}

// GenerateStoragePath generates a structured storage path based on file context
func GenerateStoragePath(cfg StoragePathConfig) string {
	now := time.Now()
	datePrefix := now.Format("2006-01-02")

	switch cfg.Context {
	case models.FileContextProfilePhoto:
		// Structure: {user_id}/profile/{hash}.{ext}
		return fmt.Sprintf("%s/profile/%s%s", cfg.UserID, cfg.ContentHash, cfg.FileExtension)

	case models.FileContextMessageMedia:
		// Structure: {user_id}/messages/{conversation_id}/{date}/{hash}.{ext}
		if cfg.ConversationID != "" {
			return fmt.Sprintf("%s/messages/%s/%s/%s%s",
				cfg.UserID, cfg.ConversationID, datePrefix, cfg.ContentHash, cfg.FileExtension)
		}
		// Fallback if no conversation ID
		return fmt.Sprintf("%s/messages/%s/%s%s",
			cfg.UserID, datePrefix, cfg.ContentHash, cfg.FileExtension)

	case models.FileContextAlbum:
		// Structure: {user_id}/albums/{album_id}/{hash}.{ext}
		if cfg.AlbumID != "" {
			return fmt.Sprintf("%s/albums/%s/%s%s",
				cfg.UserID, cfg.AlbumID, cfg.ContentHash, cfg.FileExtension)
		}
		// Fallback if no album ID
		return fmt.Sprintf("%s/albums/%s/%s%s",
			cfg.UserID, datePrefix, cfg.ContentHash, cfg.FileExtension)

	case models.FileContextSticker:
		// Structure: stickers/{pack_id}/{hash}.{ext}
		if cfg.StickerPackID != "" {
			return fmt.Sprintf("stickers/%s/%s%s",
				cfg.StickerPackID, cfg.ContentHash, cfg.FileExtension)
		}
		// Fallback for standalone stickers
		return fmt.Sprintf("stickers/standalone/%s%s", cfg.ContentHash, cfg.FileExtension)

	case models.FileContextDocument:
		// Structure: {user_id}/documents/{date}/{hash}.{ext}
		return fmt.Sprintf("%s/documents/%s/%s%s",
			cfg.UserID, datePrefix, cfg.ContentHash, cfg.FileExtension)

	case models.FileContextVoiceNote:
		// Structure: {user_id}/voice/{conversation_id}/{date}/{hash}.{ext}
		if cfg.ConversationID != "" {
			return fmt.Sprintf("%s/voice/%s/%s/%s%s",
				cfg.UserID, cfg.ConversationID, datePrefix, cfg.ContentHash, cfg.FileExtension)
		}
		return fmt.Sprintf("%s/voice/%s/%s%s",
			cfg.UserID, datePrefix, cfg.ContentHash, cfg.FileExtension)

	case models.FileContextGeneral:
		fallthrough
	default:
		// Structure: {user_id}/general/{date}/{hash}.{ext}
		return fmt.Sprintf("%s/general/%s/%s%s",
			cfg.UserID, datePrefix, cfg.ContentHash, cfg.FileExtension)
	}
}

// GenerateThumbnailPath generates a thumbnail path based on the original file path
func GenerateThumbnailPath(originalPath, size, hash, extension string) string {
	dir := filepath.Dir(originalPath)
	return fmt.Sprintf("%s/thumbnails/%s/%s%s", dir, size, hash, extension)
}

// ParseFileContext determines file context from metadata
func ParseFileContext(contextStr string) models.FileContext {
	context := models.FileContext(contextStr)
	if context.IsValid() {
		return context
	}
	return models.FileContextGeneral
}
