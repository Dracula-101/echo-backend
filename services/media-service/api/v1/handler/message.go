package handler

import (
	"fmt"
	"net/http"

	"media-service/internal/service/models"
	req "shared/server/request"
	"shared/server/response"
)

func (h *Handler) UploadMessageMedia(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()

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

	conversationID := r.FormValue("conversation_id")
	if conversationID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "conversation_id is required", fmt.Errorf("missing conversation_id"))
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
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to upload message media", err)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusCreated, "Message media uploaded successfully", output)
}
