package handler

import (
	"context"
	"fmt"
	"time"

	"auth-service/api/v1/dto"
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
)

// Login flow at a glance:
//
//	[1] Request helper — parse & validate JSON.
//	[2] LocationService.Lookup — GeoIP timezone/country for metadata.
//	[3] AuthService.GetUserByEmail — fetch user by email.
//	[4] User status check — ensure account is active and not locked.
//	[5] AuthService.Login — validate credentials and generate tokens.
//	[6] SessionService.GetSessionByUserId — check for existing active session.
//	[7] SessionService.CreateSession — create new session if none exists.
//	[8] Response helper — return 200 JSON body with tokens.
//
// Failures short-circuit immediately with structured error payloads.
func (h *AuthHandler) Login(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Login request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	loginRequest := dto.NewLoginRequest()
	if !handler.ParseValidateAndSend(loginRequest) {
		h.log.Warn("Login request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	cacheErr := h.authCache.CheckIPBlocked(ctx, handler.GetClientIP())
	if cacheErr != nil {
		if cacheErr.Code == authErrors.CodeIPBlocked {
			h.log.Warn("Blocked login attempt from IP",
				logger.String("service", authErrors.ServiceName),
				logger.String("request_id", requestID),
				logger.String("ip_address", handler.GetClientIP()),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Too many failed login attempts from this IP. Please try again later.", nil)
			return
		}
	}

	if len(loginRequest.Password) > 128 {
		h.log.Warn("Login attempt with excessively long password",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", loginRequest.Email),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Invalid credentials - please check your email and password.", nil)
		return
	}

	deviceInfo := handler.GetDeviceInfo()
	browserInfo := handler.GetBrowserInfo()
	userAgent := handler.GetUserAgent()
	locationInfo, _ := request.GetIPAddressInfoFromContext(ctx)

	h.log.Debug("Extracting request metadata",
		logger.String("service", authErrors.ServiceName),
		logger.String("device_os", deviceInfo.OS),
		logger.String("browser", browserInfo.Name),
	)

	user, authErr := h.authService.GetUserByEmail(ctx, loginRequest.Email)
	if authErr != nil {
		h.log.Error("Failed to fetch user during login", logger.Error(authErr.Error))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
		return
	}
	if user == nil {
		h.log.Warn("Login attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", loginRequest.Email),
		)
		// delay by 200ms to mitigate user enumeration attacks
		utils.SleepWithContext(ctx, time.Millisecond*200)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), fmt.Sprintf("User does not exist with email %s", loginRequest.Email), nil)
		return
	}

	if statusErr := user.CanLogin(); statusErr != nil {
		h.handleAccountStatusError(ctx, handler, requestID, user, statusErr)
		return
	}

	userResult, authErr := h.authService.Login(ctx, domain.LoginInput{
		Email:        loginRequest.Email,
		Password:     loginRequest.Password,
		DeviceInfo:   deviceInfo,
		LocationInfo: &locationInfo,
	})
	if authErr != nil {
		if authErr.Code == authErrors.CodeInvalidCredentials {
			h.recordFailedLogin(ctx, deviceInfo, &locationInfo, user.ID, userAgent, authErr.Message)
		}
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
		return
	}
	if userResult == nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), fmt.Sprintf("No user found with email %s", loginRequest.Email), nil)
		return
	}

	session := &domain.CreateSessionOutput{}
	activeSession, sessErr := h.sessionService.GetSessionByUserId(ctx, user.ID, deviceInfo.ID)
	if sessErr != nil {
		h.log.Error("Failed to fetch active session during login", logger.Error(sessErr))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login", sessErr)
		return
	}

	if activeSession == nil {
		h.log.Info("No active session found, creating new session",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userResult.User.ID),
		)
		isMobile := deviceInfo.IsMobile()
		createdSession, err := h.sessionService.CreateSession(ctx, domain.CreateSessionInput{
			UserID:          userResult.User.ID,
			RefreshToken:    userResult.RefreshToken,
			Device:          deviceInfo,
			Browser:         browserInfo,
			UserAgent:       userAgent,
			IP:              locationInfo,
			Latitude:        locationInfo.Latitude,
			Longitude:       locationInfo.Longitude,
			IsMobile:        isMobile,
			IsTrustedDevice: false,
			FCMToken:        utils.DerefString(loginRequest.FCMToken),
			APNSToken:       utils.DerefString(loginRequest.APNSToken),
			SessionType: func() domain.SessionType {
				if isMobile {
					return domain.SessionTypeMobile
				}
				return domain.SessionTypeWeb
			}(),
			ExpiresAt: userResult.ExpiresAt,
			Metadata: map[string]any{
				"request_id":     requestID,
				"correlation_id": correlationID,
				"device-id":      deviceInfo.ID,
			},
		})
		if err != nil {
			h.log.Error("Failed to create session after login", logger.Error(err))
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to create session", err)
			return
		}
		session = createdSession
	} else {
		h.log.Info("Active session found, updating session tokens",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userResult.User.ID),
			logger.String("session_id", activeSession.ID),
		)
		sessionData, dbErr := h.sessionService.UpdateSession(ctx, userResult.User.ID, activeSession.ID, domain.UpdateSession{
			FCMToken:    utils.SafePtrString(loginRequest.FCMToken),
			APNSToken:   utils.SafePtrString(loginRequest.APNSToken),
			RevokedAt:   nil,
			PushEnabled: utils.PtrBool(utils.DerefString(loginRequest.FCMToken) != "" || utils.DerefString(loginRequest.APNSToken) != ""),
		})
		if dbErr != nil {
			h.log.Error("Failed to update session tokens", logger.Error(dbErr))
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to update session tokens", dbErr)
			return
		}
		h.log.Info("Session tokens updated successfully",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", userResult.User.ID),
			logger.Any("session", sessionData),
		)
		session = &domain.CreateSessionOutput{
			SessionId:    sessionData.ID,
			SessionToken: sessionData.SessionToken,
		}
	}

	h.log.Info("Login successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", userResult.User.ID),
		logger.String("session_id", session.SessionId),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Login successful",
		dto.NewLoginResponse(
			*userResult.User,
			session.SessionId,
			session.SessionToken,
			userResult.AccessToken,
			userResult.RefreshToken,
			userResult.ExpiresAt,
		),
	)
}

func (h *AuthHandler) handleAccountStatusError(ctx context.Context, handler *req.RequestHandler, requestID string, user *domain.User, statusErr error) {
	var message string
	switch statusErr {
	case authErrors.ErrAccountDeactivated:
		message = "Account is deactivated. Please contact support."
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), message, statusErr)
	case authErrors.ErrAccountSuspended:
		message = "Account is suspended. Please contact support."
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), message, statusErr)
	case authErrors.ErrAccountLocked:
		message = "Account is locked due to multiple failed login attempts. Please reset your password or contact support."
		response.LockedError(ctx, handler.Request(), handler.Writer(), message)
	case authErrors.ErrAccountPending:
		message = "Account is pending verification. Please verify your account before logging in."
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), message, statusErr)
	case authErrors.ErrAccountDeleted:
		message = "Account has been deleted. Please contact support for further assistance."
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), message, statusErr)
	default:
		h.log.Error("Unknown account status during login",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", user.ID),
			logger.Error(statusErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login due to unknown account status", nil)
	}
}

func (h *AuthHandler) recordFailedLogin(ctx context.Context, device req.DeviceInfo, locationInfo *req.IpAddressInfo, userID string, userAgent string, failureReason string) {
	if locationInfo == nil {
		locationInfo = &req.IpAddressInfo{}
	}

	input := domain.FailedLoginAttemptInput{
		UserID:        userID,
		Device:        device,
		Location:      locationInfo,
		UserAgent:     userAgent,
		Reason:        failureReason,
		LoginMethod:   "password",
		IsNewDevice:   false,
		IsNewLocation: false,
	}

	if err := h.authService.RecordFailedLoginAttempt(ctx, input); err != nil {
		h.log.Error("Failed to record failed login attempt", logger.Error(err.Error))
	}

	h.log.Warn("Failed login attempt recorded",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("reason", failureReason),
	)
}
