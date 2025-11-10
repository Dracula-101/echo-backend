package handler

import (
	"fmt"
	"net/http"

	"media-service/api/dto"
	"media-service/internal/service/models"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

// Upload handles file upload
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	handler := request.NewHandler(r, w)

	defer func() {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	file, fileHeader, err := handler.GetFormFile("file")
	if err != nil {
		h.log.Error("Failed to get file from form", logger.Error(err))
		response.BadRequestError(ctx, r, w, "File is required", err)
		return
	}
	defer file.Close()

	visibility := "private"
	var metadata map[string]interface{}

	deviceID := handler.GetDeviceInfo().ID
	ipAddress := handler.GetClientIP()
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	userAgent := r.UserAgent()

	output, err := h.mediaService.UploadFile(ctx, models.UploadFileInput{
		UserID:      userID,
		FileReader:  file,
		FileName:    fileHeader.Filename,
		FileSize:    fileHeader.Size,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Visibility:  visibility,
		DeviceID:    deviceID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Metadata:    metadata,
	})

	if err != nil {
		h.log.Error("Failed to upload file",
			logger.String("user_id", userID),
			logger.String("file_name", fileHeader.Filename),
			logger.Error(err),
		)
		response.InternalServerError(ctx, r, w, "Failed to upload file", err)
		return
	}

	// Build response
	responseDTO := dto.UploadFileResponse{
		FileID:           output.FileID,
		FileName:         output.FileName,
		FileSize:         output.FileSize,
		FileType:         output.FileType,
		StorageURL:       output.StorageURL,
		CDNURL:           output.CDNURL,
		ProcessingStatus: output.ProcessingStatus,
		AccessToken:      output.AccessToken,
		UploadedAt:       output.UploadedAt,
	}

	h.log.Info("File uploaded successfully",
		logger.String("user_id", userID),
		logger.String("file_id", output.FileID),
	)

	response.JSONWithContext(ctx, r, w, http.StatusCreated, responseDTO)
}
