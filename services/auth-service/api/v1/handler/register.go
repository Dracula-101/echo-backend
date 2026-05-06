package handler

import (
	"auth-service/api/v1/dto"
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"strings"
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

	h.log.Debug("Checking if email is valid",
		logger.String("request_id", requestID),
		logger.String("email", reqBody.Email),
	)
	emailInfo, emailErr := utils.ValidateEmail(reqBody.Email)
	if emailErr != nil {
		h.log.Warn("Invalid email format in registration request",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", reqBody.Email),
			logger.Error(emailErr),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid email format", pkgErrors.New(authErrors.CodeInvalidEmail, emailErr.Error()))
		return
	}
	_ = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(emailInfo.LocalPart), ".", "")) + "@" + strings.ToLower(strings.TrimSpace(emailInfo.Domain))
	// TODO: Check with blocklist service to see if email domain is disposable/blacklisted and reject if so.

	// validate the phone number if provided
	// var phoneInfo *utils.PhoneInfo
	// if reqBody.PhoneNumber != nil && reqBody.PhoneCountryCode != nil {
	// 	phoneInfo, phoneErr := utils.ValidatePhoneNumber(*reqBody.PhoneNumber, *reqBody.PhoneCountryCode)
	// 	if phoneErr != nil {
	// 		h.log.Warn("Invalid phone number format in registration request",
	// 			logger.String("service", authErrors.ServiceName),
	// 			logger.String("request_id", requestID),
	// 			logger.String("phone_number", *reqBody.PhoneNumber),
	// 			logger.Error(phoneErr),
	// 		)
	// 		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Invalid phone number format", pkgErrors.New(authErrors.CodeInvalidPhoneNumber, err.Error()))
	// 		return
	// 	}
	// }

	clientIp := handler.GetClientIP()
	h.log.Debug("Checking if email is already registered",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("email", reqBody.Email),
	)
	userInfo, authErr := h.authService.IsEmailTaken(ctx, reqBody.Email)
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
	if userInfo != nil {
		switch userInfo.AccountStatus {
		case models.AccountStatusLocked:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is temporarily locked. Please try again later.", pkgErrors.New(authErrors.CodeEmailLocked, "email is temporarily locked"))
			return
		case models.AccountStatusDeleted:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email was previously registered but has been deleted. Please contact support if you believe this is an error.", pkgErrors.New(authErrors.CodeEmailDeleted, "email was previously registered but has been deleted"))
			return
		case models.AccountStatusPending:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is pending verification. Please check your inbox for the verification email.", pkgErrors.New(authErrors.CodeEmailPendingVerification, "email is pending verification"))
			return
		case models.AccountStatusActive:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is already registered", pkgErrors.New(authErrors.CodeEmailAlreadyExists, authErrors.ErrPhoneAlreadyExists.Error()))
			return
		case models.AccountStatusDeactivated:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is already registered but the account is deactivated. Please contact support.", pkgErrors.New(authErrors.CodeEmailAlreadyExists, authErrors.ErrEmailAlreadyExists.Error()))
			return
		case models.AccountStatusSuspended:
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Email is already registered but the account is suspended. Please contact support.", pkgErrors.New(authErrors.CodeEmailAlreadyExists, authErrors.ErrEmailAlreadyExists.Error()))
			return
		}
	}
	cacheErr := h.authCache.AcquireRegisterEmailLock(ctx, reqBody.Email)
	if cacheErr != nil {
		switch cacheErr.Code {
		case authErrors.CodeEmailAlreadyExists:
			h.log.Warn("Concurrent registration attempt for the same email",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("email", reqBody.Email),
			)
			response.ConflictError(ctx, handler.Request(), handler.Writer(), "Email is already being registered. Please try again later.", pkgErrors.New(authErrors.CodeEmailAlreadyExists, "email is already being registered"))
		case authErrors.CodeCacheError:
			h.log.Error("Failed to acquire email lock",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("email", reqBody.Email),
				logger.Error(cacheErr.Error),
			)
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to acquire email lock", cacheErr.Error)
		}
	}

	if reqBody.PhoneNumber != nil && reqBody.PhoneCountryCode != nil {
		h.log.Debug("Checking if phone number is already registered",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("phone_number", *reqBody.PhoneNumber),
			logger.String("phone_country_code", *reqBody.PhoneCountryCode),
		)
		userInfo, authErr = h.authService.IsPhoneTaken(ctx, *reqBody.PhoneNumber)
		if authErr != nil {
			h.log.Error("Failed to check if phone number is taken",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("phone_number", *reqBody.PhoneNumber),
				logger.String("phone_country_code", *reqBody.PhoneCountryCode),
				logger.Error(authErr.Error),
			)
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to check phone number availability", authErr.Error)
			return
		}
		if userInfo != nil {
			h.log.Debug("Phone number is already registered",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("phone_number", *reqBody.PhoneNumber),
				logger.String("phone_country_code", *reqBody.PhoneCountryCode),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Phone number is already registered", pkgErrors.New(authErrors.CodePhoneAlreadyExists, authErrors.ErrPhoneAlreadyExists.Error()))
			return
		}

		cacheErr = h.authCache.AcquireRegisterPhoneLock(ctx, *reqBody.PhoneNumber)
		if cacheErr != nil {
			switch cacheErr.Code {
			case authErrors.CodePhoneAlreadyExists:
				h.log.Warn("Concurrent registration attempt for the same phone number",
					logger.String("service", authErrors.ServiceName),
					logger.String("request_id", requestID),
					logger.String("phone_number", *reqBody.PhoneNumber),
					logger.String("phone_country_code", *reqBody.PhoneCountryCode),
				)
				response.ConflictError(ctx, handler.Request(), handler.Writer(), "Phone number is already being registered. Please try again later.", pkgErrors.New(authErrors.CodePhoneAlreadyExists, "phone number is already being registered"))
			case authErrors.CodeCacheError:
				h.log.Error("Failed to acquire phone lock",
					logger.String("service", authErrors.ServiceName),
					logger.String("request_id", requestID),
					logger.String("phone_number", *reqBody.PhoneNumber),
					logger.String("phone_country_code", *reqBody.PhoneCountryCode),
					logger.Error(cacheErr.Error),
				)
				response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to acquire phone lock", cacheErr.Error)
			}
		}
		defer func() {
			cacheErr := h.authCache.ReleaseRegisterPhoneLock(ctx, *reqBody.PhoneNumber)
			if cacheErr != nil {
				h.log.Error("Failed to release phone lock",
					logger.String("service", authErrors.ServiceName),
					logger.String("request_id", requestID),
					logger.String("phone_number", *reqBody.PhoneNumber),
					logger.String("phone_country_code", *reqBody.PhoneCountryCode),
					logger.Error(cacheErr.Error),
				)
			}
		}()
	}

	deviceInfo := handler.GetDeviceInfo()
	locationInfo, ok := request.GetIPAddressInfoFromContext(ctx)
	if !ok {
		locationInfo = request.NewIpAddressInfo()
	}
	output, authErr := h.authService.RegisterUser(ctx, domain.RegisterUserInput{
		Email:            reqBody.Email,
		Password:         reqBody.Password,
		PhoneNumber:      reqBody.PhoneNumber,
		PhoneCountryCode: reqBody.PhoneCountryCode,
		IPAddress:        clientIp,
		AcceptTerms:      reqBody.AcceptTerms,
		UserAgent:        handler.GetUserAgent(),
		Country:          locationInfo.Country,
		AppVersion:       deviceInfo.AppVersion,
	})
	if authErr != nil {
		if authErr.Code == authErrors.CodePasswordTooWeak || authErr.Code == authErrors.CodeInvalidEmail {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
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

	// sending email verification
	if output.NextStep != nil {
		if *output.NextStep == "verify_email" && output.VerificationEmailSentTo != nil {
			h.log.Info("Verification email sent",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("email", *output.VerificationEmailSentTo),
			)
			emailSendErr := h.authService.SendEmailVerification(ctx, output.UserID, output.Email, locationInfo.IP, handler.GetUserAgent())
			if emailSendErr != nil {
				h.log.Error("Failed to send email verification",
					logger.String("service", authErrors.ServiceName),
					logger.String("request_id", requestID),
					logger.String("email", *output.VerificationEmailSentTo),
					logger.Error(emailSendErr.Error),
				)
			}
		}
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Registration successful", dto.NewRegisterResponse(
		output.UserID,
		output.Email,
		output.EmailVerified,
		output.PhoneVerified,
		output.AccountStatus,
		output.NextStep,
		func() *string {
			if output.VerificationEmailSentTo != nil {
				maskedEmail := utils.MaskEmail(*output.VerificationEmailSentTo)
				return &maskedEmail
			}
			return nil
		}(),
		// send custom message if account is pending verification, otherwise generic success message
		func() string {
			if output.NextStep != nil && *output.NextStep == "verify_email" {
				return "Registration successful. Please check your email for the verification link."
			}
			return "Registration successful. You can now log in with your credentials."
		}(),
	))

	defer func() {
		cacheErr := h.authCache.ReleaseRegisterEmailLock(ctx, reqBody.Email)
		if cacheErr != nil {
			h.log.Error("Failed to release email lock",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("email", reqBody.Email),
				logger.Error(cacheErr.Error),
			)
		}
	}()
}
