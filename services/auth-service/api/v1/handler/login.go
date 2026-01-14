package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/errors"
	repositoryModels "auth-service/internal/repo/models"
	serviceModels "auth-service/internal/service/models"
	"context"
	"fmt"
	"net/http"
	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
	"shared/pkg/utils"
	req "shared/server/request"
	"shared/server/response"
	"time"
)

// Login flow at a glance:
//
//	[1] Request helper — parse & validate JSON, expose request/correlation IDs.
//	[2] AuthService.GetUserByEmail — load account + run status gates.
//	[3] LocationService.Lookup — attach GeoIP context for session metadata.
//	[4] AuthService.Login — verify credentials.
//	      -> invalid creds → LoginHistoryRepo + 400
//	      -> other auth errors → 400
//	      -> success → continue
//	[5] SessionService — reuse/create session, persist device/browser + FCM/APNs tokens.
//	[6] Response helper — return 200 JSON (user, tokens, session info).
//
// Each step logs the same request + correlation IDs for traceability.
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

	deviceInfo := handler.GetDeviceInfo()
	browserInfo := handler.GetBrowserInfo()
	userAgent := handler.GetUserAgent()
	clientIP := handler.GetClientIP()

	h.log.Debug("Extracting request metadata",
		logger.String("service", authErrors.ServiceName),
		logger.String("device_os", deviceInfo.OS),
		logger.String("browser", browserInfo.Name),
	)

	locationInfo, err := h.locationService.Lookup(clientIP)
	if err != nil {
		h.log.Error("Failed to lookup location",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("ip_address", clientIP),
			logger.Error(err),
		)
	}

	user, authErr := h.service.GetUserByEmail(ctx, loginRequest.Email)
	if authErr != nil {
		h.log.Error("Failed to fetch user during login", logger.Error(authErr))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login", authErr)
		return
	}
	if user == nil {
		h.log.Warn("Login attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", loginRequest.Email),
		)
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), fmt.Sprintf("User does not exist with email %s", loginRequest.Email), nil)
		return
	}

	switch user.AccountStatus {
	case dbModels.AccountStatusDeactivated:
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Account is deactivated. Please contact support.", nil)
		return
	case dbModels.AccountStatusSuspended:
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Account is suspended. Please contact support.", nil)
		return
	case dbModels.AccountStatusLocked:
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Account is locked due to multiple failed login attempts. Please reset your password or contact support.", nil)
		return
	case dbModels.AccountStatusPending:
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Account is pending verification. Please verify your account before logging in.", nil)
		return
	case dbModels.AccountStatusDeleted:
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), "Account has been deleted. Please contact support for further assistance.", nil)
		return
	case dbModels.AccountStatusActive:
		// Proceed with login
	default:
		h.log.Error("Unknown account status during login",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("user_id", user.ID),
			logger.String("account_status", string(user.AccountStatus)),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login due to unknown account status", nil)
		return
	}

	userResult, authErr := h.service.Login(ctx, loginRequest.Email, loginRequest.Password)
	if authErr != nil {
		if authErr.Code() == authErrors.CodeInvalidCredentials {
			h.LogFailedLogin(ctx, deviceInfo, locationInfo, user.ID, userAgent, authErr.Message())
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), authErr.Message(), authErr)
		} else {
			response.BadRequestError(ctx, handler.Request(), handler.Writer(), authErr.Message(), authErr)
		}
		return
	}
	if userResult == nil {
		response.BadRequestError(ctx, handler.Request(), handler.Writer(), fmt.Sprintf("No user found with email %s", loginRequest.Email), nil)
		return
	}

	session := &serviceModels.CreateSessionOutput{}
	activeSession, sessErr := h.sessionService.GetSessionByUserId(ctx, user.ID)
	if sessErr != nil {
		h.log.Error("Failed to fetch active session during login", logger.Error(sessErr))
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to process login", sessErr)
		return
	}

	if activeSession == nil {
		isMobile := deviceInfo.IsMobile()
		session, err = h.sessionService.CreateSession(ctx, serviceModels.CreateSessionInput{
			UserID:          userResult.User.ID,
			RefreshToken:    userResult.Session.RefreshToken,
			Device:          deviceInfo,
			Browser:         browserInfo,
			UserAgent:       userAgent,
			IP:              *locationInfo,
			Latitude:        locationInfo.Latitude,
			Longitude:       locationInfo.Longitude,
			IsMobile:        isMobile,
			IsTrustedDevice: false,
			FCMToken:        loginRequest.FCMToken,
			APNSToken:       loginRequest.APNSToken,
			SessionType: func() dbModels.SessionType {
				if isMobile {
					return dbModels.SessionTypeMobile
				}
				return dbModels.SessionTypeWeb
			}(),
			ExpiresAt: time.Unix(userResult.Session.ExpiresAt, 0),
			Metadata: map[string]interface{}{
				"request_id":     requestID,
				"correlation_id": correlationID,
			},
		})
		if err != nil {
			h.log.Error("Failed to create session after login", logger.Error(err))
			response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to create session", err)
			return
		}
	} else {
		session.SessionId = activeSession.ID
		session.SessionToken = activeSession.SessionToken
		userResult.Session.RefreshToken = *activeSession.RefreshToken
	}

	h.log.Info("Login successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", userResult.User.ID),
		logger.String("session_id", session.SessionId),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusOK, "Login successful",
		map[string]any{
			"user":          userResult.User,
			"access_token":  userResult.Session.AccessToken,
			"expires_at":    userResult.Session.ExpiresAt,
			"refresh_token": userResult.Session.RefreshToken,
			"session_token": session.SessionToken,
			"session_id":    session.SessionId,
		},
	)
}

func (h *AuthHandler) LogFailedLogin(ctx context.Context, device req.DeviceInfo, locationInfo *req.IpAddressInfo, userID string, userAgent string, failureReason string) {
	h.log.Info("Logging failed login attempt",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("ip_address", locationInfo.IP),
		logger.String("device_os", device.OS),
		logger.String("reason", failureReason),
	)

	err := h.service.LoginHistoryRepo.CreateLoginHistory(ctx, repositoryModels.CreateLoginHistoryInput{
		DeviceInfo:    device,
		IPInfo:        *locationInfo,
		FailureReason: &failureReason,
		UserID:        userID,
		SessionID:     nil,
		LoginMethod:   utils.PtrString("password"),
		Status:        utils.PtrString("failure"),
		UserAgent:     &userAgent,
		IsNewDevice:   utils.PtrBool(false),
		IsNewLocation: utils.PtrBool(false),
	})
	if err != nil {
		h.log.Error("Failed to create login history record", logger.Error(err))
	}
}
