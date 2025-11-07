package dto

import "time"

// GetFileResponse represents the response for getting a file
type GetFileResponse struct {
	FileID           string    `json:"file_id"`
	FileName         string    `json:"file_name"`
	FileSize         int64     `json:"file_size"`
	FileType         string    `json:"file_type"`
	StorageURL       string    `json:"storage_url"`
	CDNURL           string    `json:"cdn_url,omitempty"`
	ThumbnailURL     string    `json:"thumbnail_url,omitempty"`
	ProcessingStatus string    `json:"processing_status"`
	Visibility       string    `json:"visibility"`
	DownloadCount    int       `json:"download_count"`
	ViewCount        int       `json:"view_count"`
	CreatedAt        time.Time `json:"created_at"`
}

// ListFilesRequest represents the request to list files
type ListFilesRequest struct {
	FileCategory string `json:"file_category,omitempty"`
	Limit        int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset       int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	SortBy       string `json:"sort_by,omitempty" validate:"omitempty,oneof=created_at file_name file_size"`
	SortOrder    string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// ListFilesResponse represents the response for listing files
type ListFilesResponse struct {
	Files      []FileItem `json:"files"`
	TotalCount int        `json:"total_count"`
	HasMore    bool       `json:"has_more"`
}

// FileItem represents a file in a list
type FileItem struct {
	FileID       string    `json:"file_id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	FileType     string    `json:"file_type"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DeleteFileRequest represents the request to delete a file
type DeleteFileRequest struct {
	Permanent bool `json:"permanent,omitempty"`
}

// DeleteFileResponse represents the response after deleting a file
type DeleteFileResponse struct {
	Message string `json:"message"`
	FileID  string `json:"file_id"`
}
