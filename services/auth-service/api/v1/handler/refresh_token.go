package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// RefreshToken flow at a glance:
//
//	[1] Request helper — parse & validate JSON containing refresh token.
//	[2] AuthService.RefreshToken — validate token, fetch user, check status, generate new tokens.
//	[3] Response helper — return 200 JSON body with new tokens.
//
// Failures short-circuit immediately with structured error payloads.
func (h *AuthHandler) RefreshToken(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Refresh token request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	refreshRequest := dto.NewRefreshTokenRequest()
	if !handler.ParseValidateAndSend(refreshRequest) {
		h.log.Warn("Refresh token request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	result, authErr := h.authService.RefreshToken(ctx, refreshRequest.RefreshToken)
	if authErr != nil {
		if authErr.Code == authErrors.CodeTokenValidationFailed || authErr.Code == authErrors.CodeInvalidToken {
			h.log.Warn("Invalid refresh token",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.Error(authErr.Error),
			)
			response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
			return
		}
		if authErr.Code == authErrors.CodeUserNotFound {
			h.log.Warn("Token refresh for non-existent user",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.Error(authErr.Error),
			)
			response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
			return
		}
		h.log.Error("Failed to refresh token",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.Error(authErr.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
		return
	}

	if result == nil || result.User == nil {
		h.log.Error("Unexpected nil result from refresh token",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to refresh token", nil)
		return
	}

	h.log.Info("Token refresh successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", result.User.ID),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Token refreshed successfully",
		dto.NewRefreshTokenResponse(
			result.AccessToken,
			result.RefreshToken,
			result.ExpiresAt,
		),
	)
}
