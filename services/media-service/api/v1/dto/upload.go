package dto

import "time"

// UploadFileRequest represents the request to upload a file
type UploadFileRequest struct {
	Visibility string                 `json:"visibility" validate:"omitempty,oneof=private public unlisted"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// UploadFileResponse represents the response after file upload
type UploadFileResponse struct {
	FileID           string    `json:"file_id"`
	FileName         string    `json:"file_name"`
	FileSize         int64     `json:"file_size"`
	FileType         string    `json:"file_type"`
	StorageURL       string    `json:"storage_url"`
	CDNURL           string    `json:"cdn_url,omitempty"`
	ProcessingStatus string    `json:"processing_status"`
	AccessToken      string    `json:"access_token,omitempty"`
	UploadedAt       time.Time `json:"uploaded_at"`
}
