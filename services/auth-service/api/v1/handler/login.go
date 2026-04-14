package handler

import (
	"context"
	"fmt"
	"time"

	"auth-service/api/v1/dto"
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	req "shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) Login(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Login request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	loginRequest := dto.NewLoginRequest()
	if !handler.ParseValidateAndSend(loginRequest) {
		return
	}

	if cacheErr := h.authCache.CheckIPBlocked(ctx, handler.GetClientIP()); cacheErr != nil {
		if cacheErr.Code == authErrors.CodeIPBlocked {
			h.log.Warn("Blocked login attempt from IP",
				logger.String("service", authErrors.ServiceName),
				logger.String("ip_address", handler.GetClientIP()),
			)
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Too many failed login attempts from this IP. Please try again later.", nil)
			return
		}
	}

	if len(loginRequest.Password) > 128 {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Invalid credentials - please check your email and password.", nil)
		return
	}

	deviceInfo := handler.GetDeviceInfo()
	browserInfo := handler.GetBrowserInfo()
	userAgent := handler.GetUserAgent()
	locationInfo, _ := req.GetIPAddressInfoFromContext(ctx)

	user, authErr := h.authService.GetUserByEmail(ctx, loginRequest.Email)
	if authErr != nil {
		h.log.Error("Failed to fetch user during login", logger.Error(authErr.Error))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), authErr.Message, authErr.Error)
		return
	}
	if user == nil {
		// Delay to mitigate user enumeration.
		utils.SleepWithContext(ctx, time.Millisecond*200)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), fmt.Sprintf("User does not exist with email %s", loginRequest.Email), nil)
		return
	}

	// Prevent race conditions in session-limit enforcement from concurrent logins for the same user.
	acquired, lockErr := h.rateLimiter.AcquireLoginLock(ctx, user.ID)
	defer func() {
		if acquired {
			h.rateLimiter.ReleaseLoginLock(ctx, user.ID)
		}
	}()
	if lockErr != nil {
		h.log.Error("Failed to acquire login lock",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(lockErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login", lockErr)
		return
	}
	if !acquired {
		response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusConflict, "A login request for this account is already in progress. Please retry shortly.", pkgErrors.New(
			authErrors.CodeConcurrentLoginDetected,
			"Concurrent login attempt detected for user",
		))
		return
	}

	if statusErr := user.CanLogin(); statusErr != nil {
		h.handleAccountStatusError(ctx, handler, user, statusErr)
		return
	}

	userResult, authErr := h.authService.Login(ctx, domain.LoginInput{
		Email:        loginRequest.Email,
		Password:     loginRequest.Password,
		DeviceInfo:   deviceInfo,
		LocationInfo: &locationInfo,
	})
	if authErr != nil {
		h.handleFailedLogin(ctx, user.ID, handler, *authErr)
		return
	}

	// Check Trusted Device and New Location
	isTrustedDevice, trustErr := h.authService.IsDeviceTrusted(ctx, userResult.User.ID, deviceInfo.ID)
	if trustErr != nil {
		h.log.Error("Failed to determine device trust",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userResult.User.ID),
			logger.String("device_id", deviceInfo.ID),
			logger.Error(trustErr.Error),
		)
		isTrustedDevice = false
	}
	isNewLocation, locationTrustErr := h.authService.IsNewLocation(ctx, userResult.User.ID, locationInfo.City)
	if locationTrustErr != nil {
		h.log.Error("Failed to determine if login location is new",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userResult.User.ID),
			logger.String("location", locationInfo.City),
			logger.Error(locationTrustErr.Error),
		)
	}

	sessionErr := h.sessionService.PrepareSessionSlot(ctx, userResult.User.ID, deviceInfo.ID)
	if sessionErr != nil {
		h.log.Error("Failed to prepare session slot during login", logger.Error(sessionErr))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login", sessionErr)
		return
	}

	session := &domain.CreateSessionOutput{}
	activeSession, sessErr := h.sessionService.GetSessionByUserId(ctx, user.ID, deviceInfo.ID)
	if sessErr != nil {
		h.log.Error("Failed to fetch active session during login", logger.Error(sessErr))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login", sessErr)
		return
	}

	// TODO: Check for 2FA requirement based on user settings and context before creating session

	if activeSession == nil {
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
			IsTrustedDevice: isTrustedDevice,
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
		session = &domain.CreateSessionOutput{
			SessionId:    sessionData.ID,
			SessionToken: sessionData.SessionToken,
		}
	}

	h.log.Info("Login successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userResult.User.ID),
		logger.String("session_id", session.SessionId),
	)

	authErr = h.authService.RecordSuccessfulLogin(ctx, userResult.User.ID, session.SessionId, deviceInfo, locationInfo)
	if authErr != nil {
		h.log.Error("Failed to record successful login",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userResult.User.ID),
			logger.Error(authErr.Error),
		)
		// TODO:
		// auth.outbox: auth.session.created event, and if is_new_device = true: auth.new_device_login event.
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Login successful",
		dto.NewLoginResponse(
			*userResult.User,
			session.SessionId,
			session.SessionToken,
			userResult.AccessToken,
			userResult.RefreshToken,
			isNewLocation,
			isTrustedDevice,
			userResult.ExpiresAt,
		),
	)
}

func (h *AuthHandler) handleAccountStatusError(ctx context.Context, handler *req.RequestHandler, user *domain.User, statusErr error) {
	switch statusErr {
	case authErrors.ErrAccountDeactivated:
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), "Account is deactivated. Please contact support.", statusErr)
	case authErrors.ErrAccountSuspended:
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), "Account is suspended. Please contact support.", statusErr)
	case authErrors.ErrAccountLocked:
		response.LockedError(ctx, handler.Request(), handler.Writer(), "Account is locked due to multiple failed login attempts. Please reset your password or contact support.")
	case authErrors.ErrAccountPending:
		response.ForbiddenError(ctx, handler.Request(), handler.Writer(), "Account is pending verification. Please verify your account before logging in.", statusErr)
	case authErrors.ErrAccountDeleted:
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Account has been deleted. Please contact support for further assistance.", statusErr)
	default:
		h.log.Error("Unknown account status during login",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(statusErr),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login due to unknown account status", nil)
	}
}

func (h *AuthHandler) handleFailedLogin(ctx context.Context, userID string, handler *req.RequestHandler, authError authErrors.AuthError) {
	switch authError.Code {
	case authErrors.CodeInvalidCredentials:
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "Invalid email or password", authError.Error)
	case authErrors.CodeAccountLocked:
		response.LockedError(ctx, handler.Request(), handler.Writer(), "Account is locked due to multiple failed login attempts. Please reset your password or contact support.")
	case authErrors.CodeAccountRecentlyLocked:
		count, _ := h.rateLimiter.IncrementLoginUser(ctx, userID)
		if count >= 30 {
			// Distributed brute force: many IPs targeting one account — lock for 24h.
			h.rateLimiter.LockAccountExtended(ctx, userID)
			h.log.Error("Targeted brute force attack detected — account locked for 24h",
				logger.String("service", authErrors.ServiceName),
				logger.String("user_id", userID),
				logger.Int("attempts_in_window", int(count)),
			)
			// TODO: publish event to Kafka topic "auth.suspicious.activity"
			//       payload: { user_id, reason: "distributed_brute_force", attempts: count, timestamp }
		} else {
			h.rateLimiter.LockAccount(ctx, userID)
		}
		response.LockedError(ctx, handler.Request(), handler.Writer(), "Too many failed login attempts. Account is temporarily locked. Please try again later or reset your password.")
	default:
		h.log.Error("Unknown error during failed login",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(authError.Error),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process failed login attempt", authError.Error)
	}
}
