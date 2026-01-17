package handler

import (
	"errors"
	"fmt"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
)

// GetProfile flow at a glance:
//
//	[1] Request helper — ensure path param `user_id`, add request ID to context.
//	[2] UserService.GetProfile — fetch profile from Postgres.
//	[3] Branches:
//	      · record present → respond 200 JSON.
//	      · record missing → respond 400 “not found”.
//	      · repo error → respond 400 with error message.
//
// Each response includes the same trace metadata for debuggability.
func (h *UserHandler) GetProfile(handler *req.RequestHandler) {
	ctx := handler.Context()

	userID := handler.PathParam("user_id")
	if userID == "" {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "User ID is required", errors.New("user_id path parameter is missing"))
		return
	}

	h.log.Info("Getting user profile",
		logger.String("user_id", userID),
		logger.String("request_id", handler.GetRequestID()),
	)

	user, err := h.service.GetProfile(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Failed to get user profile", err)
		return
	}

	if user == nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "User not found", fmt.Errorf("no user found with ID %s", userID))
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

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Profile retrieved successfully", resp)
}
