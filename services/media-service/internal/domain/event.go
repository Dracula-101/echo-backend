package domain

import "time"

type EventType string

var (
	ProfilePhotoUploadedEvent EventType = "profile_photo_uploaded"
)

type ThumbnailURLs struct {
	Small  *string `json:"small"`
	Medium *string `json:"medium"`
	Large  *string `json:"large"`
}

type ProfilePhotoProcessedEvent struct {
	EventType EventType     `json:"event_type"`
	FileID    string        `json:"file_id"`
	UserID    string        `json:"user_id"`
	Timestamp time.Time     `json:"timestamp"`
	Thumbnail ThumbnailURLs `json:"thumbnail"`
}
