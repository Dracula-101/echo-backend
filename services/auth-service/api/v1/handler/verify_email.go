package handler

import (
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

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

	userID, ok := req.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	err := h.authService.VerifyEmail(ctx, token, userID)
	if err != nil {
		h.log.Warn("Email verification failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("error_code", string(err.Code)),
		)

		switch err.Code {
		// 409 Conflict — already verified
		case authErrors.CodeEmailAlreadyVerified:
			response.ConflictError(ctx, handler.Request(), handler.Writer(), "Email is already verified", err.Error)
			return

		// 410 Gone — token expired, prompt to resend
		case authErrors.CodeVerificationTokenExpired:
			response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusGone, err.Message, nil)
			return

		// 400 Bad Request — invalid token, not found, or too many attempts
		case authErrors.CodeVerificationNotFound,
			authErrors.CodeVerificationInvalid,
			authErrors.CodeTooManyVerificationAttempts:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
			return

		// 429 Too Many Requests (kept for completeness)
		case authErrors.CodeVerificationEmailRecentlySent:
			response.TooManyRequestsError(ctx, handler.Request(), handler.Writer(), err.Message, 0)
			return
		}

		// Fallback — 500
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to verify email", err.Error)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Email verified successfully", nil)
}
