package handler

import (
	"fmt"
	"net/http"

	"media-service/internal/service/models"
	"shared/server/request"
	"shared/server/response"
)

func (h *Handler) GetStorageStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.GetStorageStatsInput{
		UserID: userID,
	}

	output, err := h.mediaService.GetStorageStats(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to get storage stats", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, output)
}
