package handler

import (
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	"user-service/internal/service/models"
)

func (h *UserHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	createProfileRequest := dto.NewCreateProfileRequest()
	if !handler.ParseValidateAndSend(createProfileRequest) {
		return
	}

	userId, _ := request.GetUserIDFromContext(r.Context())
	location, err := h.locationService.Lookup(handler.GetClientIP())
	if err != nil {
		h.log.Error("failed to lookup location", logger.Error(err))
		response.InternalServerError(r.Context(), r, w, "Failed to lookup location", err)
		return
	}

	userName, err := h.service.GenerateUsername(r.Context(), createProfileRequest.DisplayName)
	if err != nil {
		h.log.Error("failed to generate username", logger.Error(err))
		response.InternalServerError(r.Context(), r, w, "Failed to generate username", err)
		return
	}

	profile, err := h.service.CreateProfile(r.Context(), &models.Profile{
		UserID:       userId,
		Username:     userName,
		DisplayName:  createProfileRequest.DisplayName,
		FirstName:    &createProfileRequest.FirstName,
		Bio:          &createProfileRequest.Bio,
		AvatarURL:    &createProfileRequest.AvatarURL,
		LanguageCode: &createProfileRequest.LanguageCode,
		LastName:     &createProfileRequest.LastName,
		Timezone:     &location.Timezone,
		CountryCode:  &location.Country,
		IsVerified:   false,
	})
	if err != nil {
		h.log.Error("failed to create profile", logger.String("user_id", userId), logger.Error(err))
		response.InternalServerError(r.Context(), r, w, "Failed to create profile", err)
		return
	}

	response.JSONWithMessage(r.Context(), r, w, http.StatusCreated, "Profile created successfully", dto.CreateProfileResponse{
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
