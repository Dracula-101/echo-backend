package handler

import (
	"fmt"
	"net/http"

	"media-service/internal/service/models"
	"shared/server/request"
	"shared/server/response"
)

func (h *Handler) UploadMessageMedia(w http.ResponseWriter, r *http.Request) {
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

	conversationID := r.FormValue("conversation_id")
	if conversationID == "" {
		response.BadRequestError(ctx, r, w, "conversation_id is required", fmt.Errorf("missing conversation_id"))
		return
	}

	deviceID := handler.GetDeviceInfo().ID
	ipAddress := handler.GetClientIP()
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	input := models.UploadMessageMediaInput{
		UserID:         userID,
		ConversationID: conversationID,
		MessageID:      r.FormValue("message_id"),
		FileReader:     file,
		FileName:       fileHeader.Filename,
		FileSize:       fileHeader.Size,
		ContentType:    fileHeader.Header.Get("Content-Type"),
		Caption:        r.FormValue("caption"),
		DeviceID:       deviceID,
		IPAddress:      ipAddress,
		UserAgent:      r.UserAgent(),
	}

	output, err := h.mediaService.UploadMessageMedia(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to upload message media", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusCreated, output)
}
