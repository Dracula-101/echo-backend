package handler

import (
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) VerifyPhone(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Verify phone request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	otp := handler.QueryParam("otp")
	if otp == "" {
		h.log.Warn("OTP not provided in request",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "OTP is required", nil)
		return
	}

	userID, ok := req.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	err := h.authService.VerifyPhone(ctx, userID.String(), otp)
	if err != nil {
		switch err.Code {
		case authErrors.CodeOTPNotFound, authErrors.CodeOTPExpired, authErrors.CodeOtpHashingFailed, authErrors.CodeInvalidOTP, authErrors.CodeOTPVerificationCooldown:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), err.Message, nil)
		case authErrors.CodePhoneAlreadyVerified:
			response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Phone number already verified", nil)
		case authErrors.CodeTooManyOTPAttempts:
			response.TooManyRequestsError(ctx, handler.Request(), handler.Writer(), err.Message, 10)
		case authErrors.CodeSuspiciousActivity:
			response.ForbiddenError(ctx, handler.Request(), handler.Writer(), err.Message, nil)
		case authErrors.CodeDatabaseError:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to verify phone number", nil)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Internal server error", nil)
		}
		return
	}

	h.log.Info("Phone number verified successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("user_id", userID.String()),
	)
	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Phone number verified successfully", nil)
}
