package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Forgot password flow at a glance:
//
//	[1] Request helper — parse & validate JSON (email).
//	[2] AuthService.ForgotPassword — look up user, generate reset token, send email.
//	[3] Response helper — always return 200 to prevent email enumeration.
//	[4] Failure handling — only surface email send failures; all other errors are silent.
func (h *AuthHandler) ForgotPassword(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Forgot password request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	forgotDto := dto.NewForgotPasswordRequest()
	if !handler.ParseValidateAndSend(forgotDto) {
		h.log.Warn("Forgot password request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	err := h.authService.ForgotPassword(ctx, forgotDto.Email, handler.GetClientIP(), handler.GetUserAgent())
	if err != nil {
		h.log.Error("Failed to process forgot password request",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.Error(err.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process forgot password request", err.Error)
		return
	}

	h.log.Info("Forgot password request processed successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
	)
	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Password reset instructions sent to email", nil)
}
