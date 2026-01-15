package handler

import (
	"auth-service/api/v1/dto"
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	"net/http"
	"shared/pkg/logger"
	req "shared/server/request"
	"shared/server/response"
)

// Register flow at a glance:
//
//	[1] Request helper — parse & validate JSON, capture request/correlation IDs.
//	[2] AuthService.IsEmailTaken — fast-fail if email already exists (400).
//	[3] AuthService.RegisterUser — create account, enforce password/email policy.
//	      · weak password / invalid email → 400 with message.
//	      · other issues → 500.
//	[4] Email provider (triggered inside service) — send verification mail.
//	[5] Response helper — return 201 JSON with user + verification status.
//
// Logs include request + correlation IDs for every branch.
func (h *AuthHandler) Register(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Registration request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	reqBody := dto.NewRegisterRequest()
	if ok := handler.ParseValidateAndSend(reqBody); !ok {
		h.log.Warn("Registration request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	clientIp := handler.GetClientIP()

	h.log.Debug("Checking if email is already registered",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("email", reqBody.Email),
	)

	emailTaken, authErr := h.authService.IsEmailTaken(ctx, reqBody.Email)
	if authErr != nil {
		h.log.Error("Failed to check if email is taken",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", reqBody.Email),
			logger.Error(authErr.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to check email availability", authErr.Error)
		return
	}
	if emailTaken {
		h.log.Warn("Registration attempt with existing email",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", reqBody.Email),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is already registered", nil)
		return
	}

	output, authErr := h.authService.RegisterUser(ctx, domain.RegisterUserInput{
		Email:            reqBody.Email,
		Password:         reqBody.Password,
		PhoneNumber:      reqBody.PhoneNumber,
		PhoneCountryCode: reqBody.PhoneCountryCode,
		IPAddress:        clientIp,
		UserAgent:        handler.GetUserAgent(),
		AcceptTerms:      reqBody.AcceptTerms,
	})
	if authErr != nil {
		if authErr.Code == authErrors.CodePasswordTooWeak || authErr.Code == authErrors.CodeInvalidEmail {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), authErr.Message, nil)
		} else {
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
		}
		return
	}

	h.log.Info("User registered successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", output.UserID),
		logger.String("email", output.Email),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusCreated, "Registration successful",
		dto.NewRegisterResponse(
			output.UserID,
			output.Email,
			output.EmailVerificationSent,
		),
	)
}
