package handler

import (
	"errors"
	"net/http"

	"user-service/api/dto"
	"user-service/internal/model"

	"shared/server/request"
	"shared/server/response"

	"shared/pkg/logger"
)

// GetProfile handles GET /users/:user_id
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from path parameter
	userID := request.PathParam(r, "user_id")
	if userID == "" {
		response.BadRequestError(ctx, r, w, "User ID is required", errors.New("user_id path parameter is missing"))
		return
	}

	h.log.Info("Getting user profile",
		logger.String("user_id", userID),
		logger.String("request_id", request.GetRequestID(r)),
	)

	// Get profile from service
	user, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, r, w, "Failed to fetch profile", err)
		return
	}

	// Convert to response DTO
	resp := &dto.GetProfileResponse{
		ID:           user.ID,
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Bio:          user.Bio,
		AvatarURL:    user.AvatarURL,
		LanguageCode: user.LanguageCode,
		Timezone:     user.Timezone,
		CountryCode:  user.CountryCode,
		IsVerified:   user.IsVerified,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}

	response.JSONWithMessage(ctx, r, w, http.StatusOK, "Profile retrieved successfully", resp)
}

// UpdateProfile handles PUT /users/:user_id
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from path parameter
	userID := request.PathParam(r, "user_id")
	if userID == "" {
		response.BadRequestError(ctx, r, w, "User ID is required", errors.New("user_id path parameter is missing"))
		return
	}

	h.log.Info("Updating user profile",
		logger.String("user_id", userID),
		logger.String("request_id", request.GetRequestID(r)),
	)

	// Parse request body
	var req dto.UpdateProfileRequest
	if err := request.ParseJSON(r, &req); err != nil {
		h.log.Error("Failed to parse request body",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(ctx, r, w, "Invalid request body", err)
		return
	}

	// Validate request
	if err := request.Validate(&req); err != nil {
		h.log.Error("Request validation failed",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(ctx, r, w, "Validation failed", err)
		return
	}

	// Convert to service model
	update := &model.ProfileUpdate{
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Bio:          req.Bio,
		AvatarURL:    req.AvatarURL,
		LanguageCode: req.LanguageCode,
		Timezone:     req.Timezone,
		CountryCode:  req.CountryCode,
	}

	// Update profile
	user, err := h.service.UpdateProfile(ctx, userID, update)
	if err != nil {
		h.log.Error("Failed to update profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, r, w, "Failed to update profile", err)
		return
	}

	// Convert to response DTO
	resp := &dto.UpdateProfileResponse{
		ID:           user.ID,
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Bio:          user.Bio,
		AvatarURL:    user.AvatarURL,
		LanguageCode: user.LanguageCode,
		Timezone:     user.Timezone,
		CountryCode:  user.CountryCode,
		IsVerified:   user.IsVerified,
		UpdatedAt:    user.UpdatedAt,
	}

	response.JSONWithMessage(ctx, r, w, http.StatusOK, "Profile updated successfully", resp)
}

// SearchUsers handles GET /users/search
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.log.Info("Searching users",
		logger.String("request_id", request.GetRequestID(r)),
	)

	// Get query parameters
	query := r.URL.Query().Get("query")
	if query == "" {
		response.BadRequestError(ctx, r, w, "Search query is required", errors.New("missing search query"))
		return
	}

	limit := request.GetIntQueryParam(r, "limit", 20)
	offset := request.GetIntQueryParam(r, "offset", 0)

	h.log.Debug("Search parameters",
		logger.String("query", query),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	// Search users
	users, totalCount, err := h.service.SearchUsers(ctx, query, limit, offset)
	if err != nil {
		h.log.Error("Failed to search users",
			logger.String("query", query),
			logger.Error(err),
		)
		response.InternalServerError(ctx, r, w, "Failed to search users", err)
		return
	}

	// Convert to response DTO
	searchResults := make([]dto.UserSearchResult, 0, len(users))
	for _, user := range users {
		searchResults = append(searchResults, dto.UserSearchResult{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			Bio:         user.Bio,
			IsVerified:  user.IsVerified,
		})
	}

	resp := &dto.SearchUsersResponse{
		Users:      searchResults,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	response.JSONWithMessage(ctx, r, w, http.StatusOK, "Users retrieved successfully", resp)
}
