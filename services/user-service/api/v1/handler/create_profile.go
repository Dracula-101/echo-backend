package handler

import (
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	svcmodel "user-service/internal/service/model"
)

// CreateProfile flow at a glance:
//
//	[1] Request helper — parse & validate JSON, read auth context.
//	[2] LocationService.Lookup — GeoIP timezone/country for metadata.
//	[3] UserService.GenerateUsername — derive unique handle from display name.
//	[4] UserService.CreateProfile — persist profile (Postgres) with geo hints.
//	[5] Response helper — return 201 JSON body.
//
// Failures short-circuit immediately with structured error payloads.
func (h *UserHandler) CreateProfile(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()
	w := handler.Writer()
	createProfileRequest := dto.NewCreateProfileRequest()
	if !handler.ParseValidateAndSend(createProfileRequest) {
		return
	}

	userId, _ := req.GetUserIDFromContext(ctx)
	h.log.Debug("creating profile", logger.String("user_id", userId))
	location, err := h.locationService.Lookup(handler.GetClientIP())
	if err != nil {
		h.log.Error("failed to lookup location", logger.Error(err))
		response.InternalServerError(ctx, r, w, "Failed to lookup location", err)
		return
	}

	userName, err := h.service.GenerateUsername(ctx, createProfileRequest.DisplayName)
	h.log.Debug("generated username for user",
		logger.String("username", userName),
		logger.String("user_id", userId),
	)
	if err != nil {
		h.log.Error("failed to generate username", logger.Error(err))
		response.InternalServerError(ctx, r, w, "Failed to generate username", err)
		return
	}
	h.log.Info("creating profile",
		logger.String("user_id", userId),
		logger.String("username", userName),
		logger.String("display_name", createProfileRequest.DisplayName),
		logger.String("timezone", location.Timezone),
		logger.String("country", location.Country),
		logger.String("ip", handler.GetClientIP()),
	)
	profile, err := h.service.CreateProfile(ctx, &svcmodel.Profile{
		UserID:       userId,
		Username:     userName,
		DisplayName:  createProfileRequest.DisplayName,
		FirstName:    &createProfileRequest.FirstName,
		Bio:          &createProfileRequest.Bio,
		AvatarURL:    &createProfileRequest.AvatarURL,
		LanguageCode: &createProfileRequest.LanguageCode,
		LastName:     &createProfileRequest.LastName,
		Timezone:     &location.Timezone,
		CountryCode:  &location.CountryCode,
		IsVerified:   false,
	})
	if err != nil {
		h.log.Error("failed to create profile", logger.String("user_id", userId), logger.Error(err))
		response.InternalServerError(ctx, r, w, "Failed to create profile", err)
		return
	}

	response.JSONWithMessage(ctx, r, w, http.StatusCreated, "Profile created successfully", dto.CreateProfileResponse{
		ID:           profile.ID,
		Username:     profile.Username,
		DisplayName:  *profile.DisplayName,
		FirstName:    *profile.FirstName,
		LastName:     *profile.LastName,
		Bio:          *profile.Bio,
		AvatarURL:    *profile.AvatarURL,
		LanguageCode: profile.LanguageCode,
		Timezone:     location.Timezone,
		CountryCode:  location.Country,
		IsVerified:   profile.IsVerified,
	})
}
