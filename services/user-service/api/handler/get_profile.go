package handler

import (
	"errors"
	"fmt"
	"net/http"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
	"user-service/api/dto"
)

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := request.PathParam(r, "user_id")
	if userID == "" {
		response.BadRequestError(ctx, r, w, "User ID is required", errors.New("user_id path parameter is missing"))
		return
	}

	h.log.Info("Getting user profile",
		logger.String("user_id", userID),
		logger.String("request_id", request.GetRequestID(r)),
	)

	user, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(ctx, r, w, "Failed to get user profile", err)
		return
	}

	if user == nil {
		response.BadRequestError(ctx, r, w, "User not found", fmt.Errorf("no user found with ID %s", userID))
		return
	}

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
