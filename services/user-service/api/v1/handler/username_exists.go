package handler

import (
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *UserHandler) UsernameExists(handler *req.RequestHandler) {
	ctx := handler.Context()
	correlationId := handler.GetCorrelationID()
	requestId := handler.GetRequestID()

	h.log.Info(
		"Received request to check username",
		logger.String("correlationId", correlationId),
		logger.String("requestId", requestId),
	)

	// get username from query parameters
	username := handler.QueryParam("username")
	if username == "" {
		h.log.Warn(
			"Username query parameter is missing",
			logger.String("correlationId", correlationId),
			logger.String("requestId", requestId),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Username query parameter is required", nil)
		return
	}

	exists, err := h.userService.UsernameExists(ctx, username)
	if err != nil {
		h.log.Error(
			"Error checking if username exists",
			logger.String("correlationId", correlationId),
			logger.String("requestId", requestId),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Error checking if username exists", nil)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), 200, "Username existence check successful", map[string]bool{"exists": exists})
}
