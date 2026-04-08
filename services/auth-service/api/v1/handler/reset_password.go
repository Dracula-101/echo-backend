package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Reset password flow at a glance:
//
//	[1] Request helper — parse & validate JSON (token, new_password).
//	[2] AuthService.ResetPassword — validate token, hash new password, update user, revoke sessions.
//	[3] Response helper — return 200 JSON body on success.
//	[4] Failure handling — map error codes to HTTP status and return structured payloads.
//
// Note: This endpoint does NOT require authentication. The reset token itself proves identity.
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

	authErr := h.authService.ResetPassword(ctx, resetPasswordRequest.Token, resetPasswordRequest.NewPassword, handler.GetClientIP(), handler.GetUserAgent())
	if authErr != nil {
		h.log.Warn("Password reset failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("error_code", string(authErr.Code)),
		)

		switch authErr.Code {
		case authErrors.CodePasswordResetTokenExpired:
			response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusGone, authErr.Message, nil)
			return
		case authErrors.CodePasswordResetTokenUsed:
			response.ConflictError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
			return
		case authErrors.CodeInvalidToken:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
			return
		}

		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to reset password", authErr.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Password reset successfully", nil)
}
