package handler

import (
	"net/http"
	"shared/pkg/logger"
	"shared/pkg/utils"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
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

	userId := createProfileRequest.UserID
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
	profile, err := h.service.CreateProfile(ctx, &domain.CreateProfileInput{
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
		Searchable:   true,
		IsVerified:   false,
	})
	if err != nil {
		h.log.Error("failed to create profile", logger.String("user_id", userId), logger.Error(err))
		response.InternalServerError(ctx, r, w, "Failed to create profile", err)
		return
	}

	response.JSONWithMessage(ctx, r, w, http.StatusCreated, "Profile created successfully", dto.CreateProfileResponse{
		ID:                 profile.UserID,
		Username:           profile.Username,
		DisplayName:        utils.SafeString(profile.DisplayName),
		FirstName:          utils.SafeString(profile.FirstName),
		LastName:           utils.SafeString(profile.LastName),
		Bio:                utils.SafeString(profile.Bio),
		AvatarURL:          utils.SafeString(profile.AvatarURL),
		LanguageCode:       profile.LanguageCode,
		Timezone:           utils.SafeString(profile.Timezone),
		CountryCode:        utils.SafeString(profile.CountryCode),
		AvatarThumbnailURL: utils.SafeString(profile.AvatarThumbnailURL),
		IsVerified:         profile.IsVerified,
		PhoneVerified:      profile.PhoneVerified,
		EmailVerified:      profile.EmailVerified,
		TwoFactorEnabled:   profile.TwoFactorEnabled,
		AccountStatus:      profile.AccountStatus.String(),
	})
}
