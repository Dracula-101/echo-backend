package handler

import (
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Resend verification flow at a glance:
//
//	[1] Query param — extract email from URL.
//	[2] AuthService.ResendVerification — validate user, rate-limit, generate new token, send email.
//	[3] Response helper — return 200 on success.
//	[4] Failure handling — map error codes (409 already verified, 404 not found, 429 rate limited).
func (h *AuthHandler) ResendVerification(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Resend verification request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	email := handler.QueryParam("email")
	if email == "" {
		h.log.Warn("Email is missing in resend verification request",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is required", nil)
		return
	}

	_, err := h.authService.ResendVerification(ctx, email, handler.GetClientIP(), handler.GetUserAgent())
	if err != nil {
		h.log.Warn("Resend verification failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("error_code", string(err.Code)),
		)

		switch err.Code {
		case authErrors.CodeEmailAlreadyVerified:
			response.ConflictError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
			return
		case authErrors.CodeUserNotFound:
			response.NotFoundError(ctx, handler.Request(), handler.Writer(), err.Message)
			return
		case authErrors.CodeVerificationEmailRecentlySent:
			response.TooManyRequestsError(ctx, handler.Request(), handler.Writer(), err.Message, 0)
			return
		case authErrors.CodeEmailSendFailed:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
			return
		}

		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to send verification email", err.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK,
		"Verification email sent successfully", nil)
}
