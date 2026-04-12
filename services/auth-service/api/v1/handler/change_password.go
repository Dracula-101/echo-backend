package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"auth-service/internal/repo/security_event"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Change password flow at a glance:
//
//		[1] Request helper — parse & validate JSON.
//		[2] Context extraction — get user ID from context.
//		[3] AuthService.ChangePassword — validate current password and update to new password.
//		[4] Response helper — return 200 JSON body on success.
//	 [5] Failure handling — log and return structured error payloads on failure.
//
// Note: This endpoint requires authentication and should be protected by middleware that populates the user ID in the context.
func (h *AuthHandler) ChangePassword(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Change password request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	changePasswordRequest := dto.NewChangePasswordRequest()
	if !handler.ParseValidateAndSend(changePasswordRequest) {
		h.log.Warn("Change password request validation failed",
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
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	currentPassword := changePasswordRequest.CurrentPassword
	newPassword := changePasswordRequest.NewPassword

	authErr := h.authService.ChangePassword(ctx, userID, currentPassword, newPassword, handler.GetClientIP(), handler.GetUserAgent())

	if authErr != nil {
		h.log.Warn("Change password failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(authErr.Error),
		)

		switch authErr.Code {
		case authErrors.CodeInvalidCredentials:
			h.log.Warn("Current password is incorrect",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("user_id", userID),
			)
			response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Current password is incorrect", nil)
			return
		case authErrors.CodeUserNotFound:
			h.log.Warn("User not found during password change",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("user_id", userID),
			)
			response.NotFoundError(handler.Context(), handler.Request(), handler.Writer(), "User not found")
			return
		case authErrors.CodeDatabaseError:
			h.log.Warn("Database error during password change",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("user_id", userID),
				logger.Error(authErr.Error),
			)
			response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to change password", authErr.Error)
			return
		case authErrors.CodeSamePassword:
			h.log.Warn("New password is the same as the current password",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("user_id", userID),
			)
			response.BadRequestError(handler.Context(), handler.Request(), handler.Writer(), "New password must be different from the current password", nil)
			return
		}
		response.InternalServerError(handler.Context(), handler.Request(), handler.Writer(), "Failed to change password", authErr.Error)
		return
	}

	h.log.Info("Password changed successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
	)
	sessionId, ok := req.GetSessionIDFromContext(ctx)
	if !ok {
		h.log.Warn("Session ID not found in context for security event logging",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
	}
	h.securityEventRepo.LogPasswordChangeEvent(ctx, security_event.SecurityEventInput{
		UserID: userID,
		SessionID: func() *string {
			if ok {
				return &sessionId
			} else {
				return nil
			}
		}(),
		Status:      "success",
		Description: "User changed their password successfully",
		UserAgent:   handler.GetUserAgent(),
		DeviceID:    handler.GetDeviceInfo().ID,
	})
	response.JSONWithMessage(handler.Context(), handler.Request(), handler.Writer(), response.StatusOK, "Password changed successfully", nil)
}
