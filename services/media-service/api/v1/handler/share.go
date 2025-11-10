package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"media-service/internal/service/models"
	"shared/server/request"
	"shared/server/response"

	"github.com/gorilla/mux"
)

type CreateShareRequest struct {
	FileID         string `json:"file_id" validate:"required"`
	SharedWithUser string `json:"shared_with_user_id"`
	ConversationID string `json:"conversation_id"`
	AccessType     string `json:"access_type" validate:"required"`
	ExpiresInHours int    `json:"expires_in_hours"`
	MaxViews       int    `json:"max_views"`
	Password       string `json:"password"`
}

func (h *Handler) CreateShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(ctx, r, w, "Invalid request body", err)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequestError(ctx, r, w, "Validation failed", err)
		return
	}

	var expiresIn *time.Duration
	if req.ExpiresInHours > 0 {
		duration := time.Duration(req.ExpiresInHours) * time.Hour
		expiresIn = &duration
	}

	var maxViews *int
	if req.MaxViews > 0 {
		maxViews = &req.MaxViews
	}

	input := models.CreateShareInput{
		FileID:         req.FileID,
		UserID:         userID,
		SharedWithUser: req.SharedWithUser,
		ConversationID: req.ConversationID,
		AccessType:     req.AccessType,
		ExpiresIn:      expiresIn,
		MaxViews:       maxViews,
		Password:       req.Password,
	}

	output, err := h.mediaService.CreateShare(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to create share", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusCreated, output)
}

func (h *Handler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	shareID := vars["id"]

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.RevokeShareInput{
		ShareID: shareID,
		UserID:  userID,
	}

	if err := h.mediaService.RevokeShare(ctx, input); err != nil {
		response.InternalServerError(ctx, r, w, "Failed to revoke share", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, map[string]string{"message": "share revoked successfully"})
}
