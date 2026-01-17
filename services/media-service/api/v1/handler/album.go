package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"media-service/api/v1/dto"
	"media-service/internal/service/models"
	req "shared/server/request"
	"shared/server/response"
)

func (h *Handler) CreateAlbum(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var req dto.CreateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid request body", err)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Validation failed", err)
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
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to create album", err)
		return
	}

	response.JSONWithContext(ctx, handler.Request(), handler.Writer(), http.StatusCreated, output)
}

func (h *Handler) GetAlbum(handler *req.RequestHandler) {
	ctx := handler.Context()
	albumID := handler.PathParam("id")

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.GetAlbumInput{
		AlbumID: albumID,
		UserID:  userID,
	}

	output, err := h.mediaService.GetAlbum(ctx, input)
	if err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to get album", err)
		return
	}

	response.JSONWithContext(ctx, handler.Request(), handler.Writer(), response.StatusOK, output)
}

func (h *Handler) ListAlbums(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
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
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to list albums", err)
		return
	}

	result := map[string]interface{}{
		"albums": albums,
		"limit":  limit,
		"offset": offset,
	}

	response.JSONWithContext(ctx, handler.Request(), handler.Writer(), response.StatusOK, result)
}

func (h *Handler) AddFileToAlbum(handler *req.RequestHandler) {
	ctx := handler.Context()
	albumID := handler.PathParam("id")
	r := handler.Request()

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	var req dto.AddFileToAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid request body", err)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Validation failed", err)
		return
	}

	input := models.AddFileToAlbumInput{
		AlbumID:      albumID,
		FileID:       req.FileID,
		UserID:       userID,
		DisplayOrder: req.DisplayOrder,
	}

	if err := h.mediaService.AddFileToAlbum(ctx, input); err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to add file to album", err)
		return
	}

	response.JSONWithContext(ctx, handler.Request(), handler.Writer(), response.StatusOK, map[string]string{"message": "file added to album successfully"})
}

func (h *Handler) RemoveFileFromAlbum(handler *req.RequestHandler) {
	ctx := handler.Context()
	albumID := handler.PathParam("id")
	fileID := handler.PathParam("file_id")

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User not authenticated", fmt.Errorf("missing user ID"))
		return
	}

	input := models.RemoveFileFromAlbumInput{
		AlbumID: albumID,
		FileID:  fileID,
		UserID:  userID,
	}

	if err := h.mediaService.RemoveFileFromAlbum(ctx, input); err != nil {
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to remove file from album", err)
		return
	}

	response.JSONWithContext(ctx, handler.Request(), handler.Writer(), response.StatusOK, map[string]string{"message": "file removed from album successfully"})
}
