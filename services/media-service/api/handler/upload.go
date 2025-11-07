package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"media-service/api/dto"
	"media-service/internal/service/models"

	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

// Upload handles file upload
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	if err := r.ParseMultipartForm(h.cfg.Security.MaxBodySize); err != nil {
		h.log.Error("Failed to parse multipart form", logger.Error(err))
		response.BadRequestError(ctx, r, w, "Invalid form data", err)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.log.Error("Failed to get file from form", logger.Error(err))
		response.BadRequestError(ctx, r, w, "File is required", err)
		return
	}
	defer file.Close()

	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = "private"
	}

	// Get metadata if provided
	var metadata map[string]interface{}
	if metadataStr := r.FormValue("metadata"); metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			h.log.Warn("Invalid metadata format", logger.Error(err))
		}
	}

	// Get request metadata
	deviceID := request.GetDeviceInfo(r).ID
	ipAddress := request.GetClientIP(r)
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}
	userAgent := r.UserAgent()

	// Call service
	output, err := h.service.UploadFile(ctx, models.UploadFileInput{
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
