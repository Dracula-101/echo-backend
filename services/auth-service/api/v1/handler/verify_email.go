package handler

import (
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
)

// Verify email flow at a glance:
//
//	[1] Query param — extract verification token from URL.
//	[2] Context extraction — get user ID from context.
//	[3] AuthService.VerifyEmail — validate token, mark email verified, activate user.
//	[4] Response helper — return 200 JSON body on success.
//	[5] Failure handling — map error codes (409 already verified, 410 expired, 400 invalid/too many attempts).
func (h *AuthHandler) VerifyEmail(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Verify email request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	token := handler.QueryParam("token")
	if token == "" {
		h.log.Warn("Email verification token is missing",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email verification token is required", nil)
		return
	}

	deviceInfo := handler.GetDeviceInfo()
	locationInfo, ok := request.GetIPAddressInfoFromContext(ctx)
	if !ok {
		h.log.Warn("Failed to extract location info from context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		locationInfo = request.NewIpAddressInfo()
	}
	err := h.authService.VerifyEmail(ctx, token, deviceInfo, locationInfo)
	if err != nil {
		h.log.Warn("Email verification failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("error_code", string(err.Code)),
		)

		switch err.Code {
		case authErrors.CodeEmailAlreadyVerified:
			response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Email is already verified", nil)
			return

		case authErrors.CodeVerificationTokenExpired:
			response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusGone, err.Message, nil)
			return

		case authErrors.CodeVerificationNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
			return

		case authErrors.CodeVerificationInvalid,
			authErrors.CodeTooManyVerificationAttempts:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
			return

		case authErrors.CodeVerificationEmailRecentlySent:
			response.TooManyRequestsError(ctx, handler.Request(), handler.Writer(), err.Message, 0)
			return

		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to verify email", err.Error)
			return
		}
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Email verified successfully", nil)
}
