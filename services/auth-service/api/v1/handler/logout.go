package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Logout flow at a glance:
//
//	[1] Request helper — parse & validate JSON (session_id, refresh_token).
//	[2] Context extraction — get user ID from context.
//	[3] AuthService.Logout — revoke the session and invalidate cached token.
//	[4] Response helper — return 200 JSON body on success.
//	[5] Failure handling — log and return structured error payloads on failure.
func (h *AuthHandler) Logout(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Logout request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	logoutRequest := dto.NewLogoutRequest()
	if !handler.ParseValidateAndSend(logoutRequest) {
		h.log.Warn("Logout request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	sessionId, ok := req.GetSessionIDFromContext(ctx)
	if !ok {
		h.log.Warn("Session ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	authErr := h.authService.Logout(ctx, userID, sessionId, handler.GetClientIP(), handler.GetUserAgent(), logoutRequest.RevokeAllDevices)
	if authErr != nil {
		h.log.Warn("Logout failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(authErr.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to logout", authErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK,
		func() string {
			if logoutRequest.RevokeAllDevices {
				return "Successfully logged out from all devices"
			}
			return "Successfully logged out"
		}(),
		nil,
	)
}
