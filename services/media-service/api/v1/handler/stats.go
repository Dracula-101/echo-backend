package handler

import (
	"fmt"

	"media-service/internal/service/models"
	req "shared/server/request"
	"shared/server/response"
)

func (h *Handler) GetStorageStats(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.GetStorageStatsInput{
		UserID: userID,
	}

	output, err := h.mediaService.GetStorageStats(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to get storage stats", err)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Storage stats retrieved successfully", output)
}
