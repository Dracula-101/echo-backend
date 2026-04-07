package handler

import (
	"auth-service/api/v1/dto"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) ForgotPassword(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Info("Forgot password request received",
		logger.String("requestId", requestId),
		logger.String("correlationId", correlationId),
	)

	forgotDto := dto.NewForgotPasswordRequest()
	if !handler.ParseValidateAndSend(forgotDto) {
		h.log.Warn("Failed to parse and validate forgot password request",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
		)
		return
	}

	err := h.authService.ForgotPassword(ctx, forgotDto.Email, handler.GetClientIP(), handler.GetUserAgent())
	if err != nil {
		h.log.Error("Failed to process forgot password request",
			logger.String("requestId", requestId),
			logger.String("correlationId", correlationId),
			logger.Error(err.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process forgot password request", err.Error)
		return
	}

	h.log.Info("Forgot password request processed successfully",
		logger.String("requestId", requestId),
		logger.String("correlationId", correlationId),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Password reset instructions sent to email", nil)

}
