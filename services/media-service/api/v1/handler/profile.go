package handler

import (
	"context"
	"fmt"
	"net/http"

	"media-service/internal/domain"
	"media-service/internal/service/models"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *Handler) UploadProfilePhoto(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	file, fileHeader, err := handler.GetFormFile("file")
	if err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "File is required", err)
		return
	}
	defer file.Close()

	deviceID := handler.GetDeviceInfo().ID
	ipAddress := handler.GetClientIP()

	input := models.UploadProfilePhotoInput{
		UserID:      userID,
		FileReader:  file,
		FileName:    fileHeader.Filename,
		FileSize:    fileHeader.Size,
		ContentType: fileHeader.Header.Get("Content-Type"),
		DeviceID:    deviceID,
		IPAddress:   ipAddress,
		UserAgent:   handler.GetUserAgent(),
	}

	output, err := h.mediaService.UploadProfilePhoto(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to upload profile photo", err)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusCreated, "Profile photo uploaded successfully", output)
	defer func() {
		h.log.Info("Starting async image processing for profile photo",
			logger.String("file_id", output.FileID))
		go h.mediaService.ProcessImageFile(context.Background(), output.FileID, h.mediaProcessor, domain.ProfilePhotoUploadedEvent)
	}()
}
