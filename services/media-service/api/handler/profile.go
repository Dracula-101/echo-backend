package handler

import (
	"fmt"
	"net/http"

	"media-service/internal/service/models"
	"shared/server/request"
	"shared/server/response"
)

func (h *Handler) UploadProfilePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	handler := request.NewHandler(r, w)

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	file, fileHeader, err := handler.GetFormFile("file")
	if err != nil {
		response.BadRequestError(ctx, r, w, "File is required", err)
		return
	}
	defer file.Close()

	deviceID := handler.GetDeviceInfo().ID
	ipAddress := handler.GetClientIP()
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	input := models.UploadProfilePhotoInput{
		UserID:      userID,
		FileReader:  file,
		FileName:    fileHeader.Filename,
		FileSize:    fileHeader.Size,
		ContentType: fileHeader.Header.Get("Content-Type"),
		DeviceID:    deviceID,
		IPAddress:   ipAddress,
		UserAgent:   r.UserAgent(),
	}

	output, err := h.mediaService.UploadProfilePhoto(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to upload profile photo", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusCreated, output)
}
