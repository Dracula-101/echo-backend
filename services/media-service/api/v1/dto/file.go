package dto

import (
	"shared/server/request"
	"time"

	"github.com/go-playground/validator/v10"
)

// GetFileResponse represents the response for getting a file
type GetFileResponse struct {
	FileID           string `json:"file_id"`
	FileName         string `json:"file_name"`
	FileSize         int64  `json:"file_size"`
	FileType         string `json:"file_type"`
	StorageURL       string `json:"storage_url"`
	CDNURL           string `json:"cdn_url,omitempty"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
	ProcessingStatus string `json:"processing_status"`
	Visibility       string `json:"visibility"`
	DownloadCount    int    `json:"download_count"`
	ViewCount        int    `json:"view_count"`
}

type ListFilesRequest struct {
	FileCategory string `json:"file_category,omitempty"`
	Limit        int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset       int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	SortBy       string `json:"sort_by,omitempty" validate:"omitempty,oneof=created_at file_name file_size"`
	SortOrder    string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

type ListFilesResponse struct {
	Files      []FileItem `json:"files"`
	TotalCount int        `json:"total_count"`
	HasMore    bool       `json:"has_more"`
}

type FileItem struct {
	FileID       string    `json:"file_id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	FileType     string    `json:"file_type"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type DeleteFileRequest struct {
	Permanent bool `json:"permanent,omitempty"`
}

func NewDeleteFileRequest() *DeleteFileRequest {
	return &DeleteFileRequest{}
}

func (r *DeleteFileRequest) GetValue() interface{} {
	return r
}

func (r *DeleteFileRequest) ValidateErrors(ve validator.ValidationErrors) ([]request.ValidationErrorDetail, error) {
	var errors []request.ValidationErrorDetail
	return errors, nil
}

type DeleteFileResponse struct {
	Message string `json:"message"`
	FileID  string `json:"file_id"`
}
