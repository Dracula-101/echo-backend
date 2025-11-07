package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"media-service/api/dto"
	mediaErrors "media-service/internal/errors"
	"media-service/internal/service/models"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/gorilla/mux"
)

// GetFile handles getting a file's metadata
func (h *MediaHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	fileID := vars["file_id"]

	if fileID == "" {
		response.BadRequestError(ctx, r, w, "File ID is required", fmt.Errorf(mediaErrors.CodeFileNotFound))
		return
	}

	userID, _ := request.GetUserIDFromContext(ctx)
	accessToken := request.GetAuthToken(r)

	output, err := h.service.GetFile(ctx, models.GetFileInput{
		FileID:      fileID,
		UserID:      userID,
		AccessToken: accessToken,
	})

	if err != nil {
		h.log.Error("Failed to get file",
			logger.String("file_id", fileID),
			logger.Error(err),
		)
		response.NotFoundError(ctx, r, w, "File")
		return
	}

	responseDTO := dto.GetFileResponse{
		FileID:           output.FileID,
		FileName:         output.FileName,
		FileSize:         output.FileSize,
		FileType:         output.FileType,
		StorageURL:       output.StorageURL,
		CDNURL:           output.CDNURL,
		ThumbnailURL:     output.ThumbnailURL,
		ProcessingStatus: output.ProcessingStatus,
		Visibility:       output.Visibility,
		DownloadCount:    output.DownloadCount,
		ViewCount:        output.ViewCount,
		CreatedAt:        output.CreatedAt,
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, responseDTO)
}

// DeleteFile handles file deletion
func (h *MediaHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	fileID := vars["file_id"]

	if fileID == "" {
		response.BadRequestError(ctx, r, w, "File ID is required", fmt.Errorf(mediaErrors.CodeFileNotFound))
		return
	}

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	// Parse request body for permanent deletion flag
	var req dto.DeleteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to soft delete if no body provided
		req.Permanent = false
	}

	err := h.service.DeleteFile(ctx, models.DeleteFileInput{
		FileID:    fileID,
		UserID:    userID,
		Permanent: req.Permanent,
	})

	if err != nil {
		h.log.Error("Failed to delete file",
			logger.String("file_id", fileID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, r, w, "Failed to delete file", err)
		return
	}

	responseDTO := dto.DeleteFileResponse{
		Message: "File deleted successfully",
		FileID:  fileID,
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, responseDTO)
}
