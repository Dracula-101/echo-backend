package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
)

func (h *UserHandler) GetMyProfile(handler *req.RequestHandler) {
	ctx := handler.Context()
	r := handler.Request()
	w := handler.Writer()
	userId, ok := request.GetUserIDFromContext(ctx)

	if !ok {
		h.log.Error("Failed to get user ID from context")
		response.UnauthorizedError(ctx, r, w, "Missing or invalid authentication token", nil)
		return
	}

	h.log.Debug("Received GetMyProfile request",
		logger.String("user_id", userId),
	)
	profile, err := h.service.GetProfile(ctx, userId)
	if err != nil {
		h.log.Error("Failed to get user profile", logger.Error(err))
		response.InternalServerError(ctx, r, w, "Failed to get user profile", nil)
		return
	}

	response.JSONWithMessage(ctx, r, w, 200, "User profile retrieved successfully", profile)
}
