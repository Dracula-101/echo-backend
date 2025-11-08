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
