package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) ResetPassword(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Reset password request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	resetPasswordRequest := dto.NewResetPasswordRequest()
	if !handler.ParseValidateAndSend(resetPasswordRequest) {
		h.log.Warn("Reset password request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	userId, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context for reset password",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	user, authErr := h.authService.GetUserByID(ctx, userId)
	if authErr != nil {
		h.log.Warn("Failed to get user for password reset",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userId),
			logger.Error(authErr.Error),
		)
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to process request", nil)
		return
	}

	authErr = h.authService.ResetPassword(ctx, resetPasswordRequest.Token, resetPasswordRequest.NewPassword)
	if authErr != nil {
		h.log.Warn("Password reset failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userId),
			logger.Error(authErr.Error),
		)
		response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
		return
	}

	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), response.StatusOK, "Password reset successfully", map[string]string{"email": user.Email})
}
