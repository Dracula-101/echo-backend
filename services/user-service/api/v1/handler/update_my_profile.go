package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
)

func (h *UserHandler) UpdateMyProfile(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Debug("Received UpdateMyProfile request",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)

	updateProfileRequest := dto.NewUpdateProfileRequest()
	if ok := handler.ParseValidateAndSend(updateProfileRequest); !ok {
		h.log.Error("Failed to parse and validate UpdateMyProfile request",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		return
	}

	userId, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Error("User ID not found in context",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	_, err := h.service.UpdateProfile(ctx, userId, &domain.UpdateProfileInput{})
	if err != nil {
		h.log.Error("Failed to update profile",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.Error(err),
		)
		return
	}
}
