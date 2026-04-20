package handler

import (
	"fmt"
	"net/http"
	"shared/pkg/logger"
	"shared/pkg/utils"
	req "shared/server/request"
	"shared/server/response"
	"strconv"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
)

// SearchProfiles flow at a glance:
//
//	[1] Request helper — parse & validate query params, add request ID to context.
//	[2] UserService.SearchProfiles — query Postgres with pagination.
//	[3] Branches:
//	      · results found → respond 200 JSON with profiles + metadata.
//	      · no results → respond 200 with empty list + metadata.
//	      · repo error → respond 400 with error message.
//
// Each response includes the same trace metadata for debuggability.
func (h *UserHandler) SearchProfiles(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()
	w := handler.Writer()

	// get from query params
	query := handler.QueryParam("query")
	if query == "" {
		response.BadRequestError(ctx, r, w, "Query parameter is required", fmt.Errorf("query parameter is missing"))
		return
	}
	typeStr := handler.QueryParamDefault("type", "all")
	limitStr := handler.QueryParamDefault("limit", "20")
	offsetStr := handler.QueryParamDefault("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		response.BadRequestError(ctx, r, w, "Invalid limit parameter", fmt.Errorf("limit must be a positive integer"))
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		response.BadRequestError(ctx, r, w, "Invalid offset parameter", fmt.Errorf("offset must be a non-negative integer"))
		return
	}

	queryType := domain.UserQueryType(typeStr)
	if !queryType.IsValid() {
		// If the type is invalid, we can choose to return an error or default to "all".
		queryType = domain.UserQueryTypeAll
	}

	h.log.Debug("searching user profiles",
		logger.String("query", query),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
		logger.String("request_id", handler.GetRequestID()),
	)

	profiles, total, err := h.searchService.SearchProfiles(ctx, query, queryType, limit, offset)
	if err != nil {
		h.log.Error("Failed to search profiles",
			logger.String("query", query),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Failed to search user profiles", err)
		return
	}

	users := make([]dto.UserSearchResult, len(profiles))
	for i, profile := range profiles {
		users[i] = dto.UserSearchResult{
			UserID:             profile.UserID,
			Username:           profile.Username,
			DisplayName:        utils.DerefString(profile.DisplayName),
			AvatarURL:          profile.AvatarURL,
			AvatarThumbnailURL: profile.AvatarThumbnailURL,
		}
	}

	resp := &dto.SearchUsersResponse{
		Users:      users,
		TotalCount: total,
		Limit:      limit,
		Offset:     offset,
	}

	response.JSONWithMessage(ctx, r, w, http.StatusOK, "User profiles retrieved successfully", resp)
}
