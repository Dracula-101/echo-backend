package handler

import (
	"fmt"
	"net/http"

	"media-service/api/v1/dto"
	mediaErrors "media-service/internal/errors"
	"media-service/internal/service/models"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

// GetFile handles getting a file's metadata
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	handler := request.NewHandler(r, w)
	fileID := handler.PathParam("file_id")

	if fileID == "" {
		response.BadRequestError(ctx, r, w, "File ID is required", fmt.Errorf(mediaErrors.CodeFileNotFound))
		return
	}

	userID, _ := request.GetUserIDFromContext(ctx)
	accessToken := handler.GetAuthToken()

	output, err := h.mediaService.GetFile(ctx, models.GetFileInput{
		FileID:         fileID,
		UserID:         userID,
		AccessToken:    accessToken,
		AllowedFormats: h.cfg.Features.ImageProcessing.AllowedFormats,
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
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	handler := request.NewHandler(r, w).AllowEmptyBody()

	fileID := handler.PathParam("file_id")
	if fileID == "" {
		response.BadRequestError(ctx, r, w, "File ID is required", fmt.Errorf(mediaErrors.CodeFileNotFound))
		return
	}

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var req dto.DeleteFileRequest
	handler = handler.WithMaxBodySize(h.cfg.Security.MaxBodySize).WithAllowUnknown()
	if ok := handler.ParseValidateAndSend(&req); !ok {
		h.log.Error("Failed to parse delete file request body",
			logger.String("file_id", fileID),
			logger.String("user_id", userID),
		)
		return
	}

	err := h.mediaService.DeleteFile(ctx, models.DeleteFileInput{
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
