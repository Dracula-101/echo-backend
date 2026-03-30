package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"media-service/api/v1/dto"
	"media-service/internal/service/models"
	req "shared/server/request"
	"shared/server/response"
)

func (h *Handler) CreateShare(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var reqBody dto.CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid request body", err)
		return
	}

	if err := h.validator.Struct(reqBody); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Validation failed", err)
		return
	}

	var expiresIn *time.Duration
	if reqBody.ExpiresInHours > 0 {
		duration := time.Duration(reqBody.ExpiresInHours) * time.Hour
		expiresIn = &duration
	}

	var maxViews *int
	if reqBody.MaxViews > 0 {
		maxViews = &reqBody.MaxViews
	}

	input := models.CreateShareInput{
		FileID:         reqBody.FileID,
		UserID:         userID,
		SharedWithUser: reqBody.SharedWithUser,
		ConversationID: reqBody.ConversationID,
		AccessType:     reqBody.AccessType,
		ExpiresIn:      expiresIn,
		MaxViews:       maxViews,
		Password:       reqBody.Password,
	}

	output, err := h.mediaService.CreateShare(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to create share", err)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusCreated, "Share created successfully", output)
}

func (h *Handler) RevokeShare(handler *req.RequestHandler) {
	ctx := handler.Context()
	shareID := handler.PathParam("id")

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.RevokeShareInput{
		ShareID: shareID,
		UserID:  userID,
	}

	if err := h.mediaService.RevokeShare(ctx, input); err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to revoke share", err)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Share revoked successfully", map[string]string{"message": "share revoked successfully"})
}
