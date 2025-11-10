package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"media-service/internal/service/models"
	"shared/server/request"
	"shared/server/response"

	"github.com/gorilla/mux"
)

type CreateAlbumRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description"`
	AlbumType   string `json:"album_type" validate:"required"`
	Visibility  string `json:"visibility" validate:"required"`
}

type AddFileToAlbumRequest struct {
	FileID       string `json:"file_id" validate:"required"`
	DisplayOrder int    `json:"display_order"`
}

func (h *Handler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var req CreateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(ctx, r, w, "Invalid request body", err)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequestError(ctx, r, w, "Validation failed", err)
		return
	}

	input := models.CreateAlbumInput{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		AlbumType:   req.AlbumType,
		Visibility:  req.Visibility,
	}

	output, err := h.mediaService.CreateAlbum(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to create album", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusCreated, output)
}

func (h *Handler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	albumID := vars["id"]

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.GetAlbumInput{
		AlbumID: albumID,
		UserID:  userID,
	}

	output, err := h.mediaService.GetAlbum(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to get album", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, output)
}

func (h *Handler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	input := models.ListAlbumsInput{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	}

	albums, err := h.mediaService.ListAlbums(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, r, w, "Failed to list albums", err)
		return
	}

	result := map[string]interface{}{
		"albums": albums,
		"limit":  limit,
		"offset": offset,
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, result)
}

func (h *Handler) AddFileToAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	albumID := vars["id"]

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var req AddFileToAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(ctx, r, w, "Invalid request body", err)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequestError(ctx, r, w, "Validation failed", err)
		return
	}

	input := models.AddFileToAlbumInput{
		AlbumID:      albumID,
		FileID:       req.FileID,
		UserID:       userID,
		DisplayOrder: req.DisplayOrder,
	}

	if err := h.mediaService.AddFileToAlbum(ctx, input); err != nil {
		response.InternalServerError(ctx, r, w, "Failed to add file to album", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, map[string]string{"message": "file added to album successfully"})
}

func (h *Handler) RemoveFileFromAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	albumID := vars["id"]
	fileID := vars["file_id"]

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, r, w, "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.RemoveFileFromAlbumInput{
		AlbumID: albumID,
		FileID:  fileID,
		UserID:  userID,
	}

	if err := h.mediaService.RemoveFileFromAlbum(ctx, input); err != nil {
		response.InternalServerError(ctx, r, w, "Failed to remove file from album", err)
		return
	}

	response.JSONWithContext(ctx, r, w, http.StatusOK, map[string]string{"message": "file removed from album successfully"})
}
