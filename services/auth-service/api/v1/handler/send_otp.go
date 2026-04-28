package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) SendOTP(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Send OTP request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	dto := dto.NewSendOTPRequest()
	if ok := handler.ParseValidateAndSend(dto); !ok {
		h.log.Warn("Send OTP request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	userID, ok := request.GetUserIDUUIDFromContext(ctx)
	if !ok {
		h.log.Warn("User ID not found in context",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Unauthorized", nil)
		return
	}

	err := h.authService.SendOtpVerification(ctx, userID.String(), dto.PhoneNumber, handler.GetClientIP(), handler.GetUserAgent())
	if err != nil {
		switch err.Code {
		case authErrors.CodePhoneAlreadyVerified, authErrors.CodeOTPNotFound, authErrors.CodeOTPExpired, authErrors.CodeInvalidOTP:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), err.Message, nil)
		case authErrors.CodeDatabaseError, authErrors.CodeSmsSendFailed:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), err.Message, err.Error)
		default:
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to send OTP", nil)
		}
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "OTP sent successfully", nil)
}
