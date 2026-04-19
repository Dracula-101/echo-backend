package handler

import (
	"fmt"
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
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
	if profile == nil {
		h.log.Error("User profile not found - this should not happen as userID is from auth token",
			logger.String("user_id", userId),
		)
		response.NotFoundError(ctx, r, w, "User profile not found")
		return
	}

	response.JSONWithMessage(ctx, r, w, 200, "User profile retrieved successfully", dto.NewGetProfileResponse(profile))
}

func (h *UserHandler) GetProfile(handler *req.RequestHandler) {
	ctx := handler.Context()
	correlationId := handler.GetCorrelationID()
	requestId := handler.GetRequestID()
	h.log.Info("Handling GetProfile request",
		logger.String("correlation_id", correlationId),
		logger.String("request_id", requestId),
	)
	r := handler.Request()
	w := handler.Writer()

	username := handler.PathParam("username")
	if username == "" {
		h.log.Error("Username path parameter is missing")
		response.BadRequestError(ctx, r, w, "Username path parameter is required", nil)
		return
	}

	currentUserID, ok := request.GetUserIDFromContext(ctx) // currentUserID may be empty if unauthenticated, service layer should handle it
	if !ok {
		h.log.Info("No user ID in context, treating as unauthenticated request")
		response.UnauthorizedError(ctx, r, w, "Missing or invalid authentication token", nil)
		return
	}

	h.log.Debug("Received GetProfile request",
		logger.String("username", username),
	)
	profile, err := h.service.GetProfileByUsername(ctx, currentUserID, username)
	if err != nil {
		h.log.Error("Failed to get user profile", logger.Error(err))
		response.InternalServerError(ctx, r, w, "Failed to get user profile", nil)
		return
	}
	if profile == nil {
		h.log.Info("User profile not found",
			logger.String("username", username),
		)
		response.NotFoundError(ctx, r, w, fmt.Sprintf("User profile with username '%s' not found", username))
		return
	}

	response.JSONWithMessage(ctx, r, w, response.StatusOK, "User profile retrieved successfully", dto.NewGetProfileResponse(profile))
}
