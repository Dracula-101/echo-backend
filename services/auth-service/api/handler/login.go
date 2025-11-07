package handler

import (
	"auth-service/api/dto"
	authErrors "auth-service/internal/errors"
	repositoryModels "auth-service/internal/repo/models"
	serviceModels "auth-service/internal/service/models"
	"context"
	"fmt"
	"net/http"
	"shared/pkg/database/postgres/models"
	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) LogFailedLogin(ctx context.Context, device request.DeviceInfo, locationInfo *request.IpAddressInfo, userID string, userAgent string, failureReason string) {
	h.log.Info("Logging failed login attempt",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("ip_address", locationInfo.IP),
		logger.String("device_os", device.OS),
		logger.String("reason", failureReason),
	)

	err := h.service.LoginHistoryRepo.CreateLoginHistory(ctx, repositoryModels.CreateLoginHistoryInput{
		DeviceFingerprint: h.sessionService.GenerateDeviceFingerprint(device.ID, device.OS, device.Name),
		DeviceInfo:        device,
		IPInfo:            *locationInfo,
		UserID:            userID,
		SessionID:         nil,
		LoginMethod:       utils.PtrString("password"),
		Status:            utils.PtrString("failure"),
		UserAgent:         &userAgent,
		IsNewDevice:       utils.PtrBool(false),
		IsNewLocation:     utils.PtrBool(false),
	})
	if err != nil {
		h.log.Error("Failed to create login history record",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
	}

	err = h.service.SecurityEventRepo.LogSecurityEvent(ctx, &models.SecurityEvent{
		UserID:          &userID,
		EventType:       models.SecurityEventLoginFailed,
		EventCategory:   utils.PtrString("authentication"),
		Severity:        models.SecuritySeverityMedium,
		Status:          utils.PtrString("failed"),
		Description:     utils.PtrString(fmt.Sprintf("User login attempt failed: %s", failureReason)),
		IPAddress:       &locationInfo.IP,
		UserAgent:       &userAgent,
		DeviceID:        &device.ID,
		LocationCountry: &locationInfo.Country,
		LocationCity:    &locationInfo.City,
		IsSuspicious:    false,
		Metadata:        nil,
	})
	if err != nil {
		h.log.Error("Failed to log security event",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
	}
}

func (h *AuthHandler) LogSuccessfulLogin(ctx context.Context, session *serviceModels.CreateSessionOutput, device request.DeviceInfo, locationInfo *request.IpAddressInfo, userID string, userAgent string, failureReason string) {
	h.log.Info("Logging successful login",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("session_id", session.SessionId),
		logger.String("ip_address", locationInfo.IP),
		logger.String("device_os", device.OS),
	)

	err := h.service.LoginHistoryRepo.CreateLoginHistory(ctx, repositoryModels.CreateLoginHistoryInput{
		DeviceFingerprint: session.DeviceFingerprint,
		DeviceInfo:        device,
		IPInfo:            *locationInfo,
		UserID:            userID,
		SessionID:         &session.SessionId,
		LoginMethod:       utils.PtrString("password"),
		Status:            utils.PtrString("success"),
		UserAgent:         &userAgent,
		IsNewDevice:       utils.PtrBool(false),
		IsNewLocation:     utils.PtrBool(false),
	})
	if err != nil {
		h.log.Error("Failed to create login history record",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
	}

	err = h.service.SecurityEventRepo.LogSecurityEvent(ctx, &models.SecurityEvent{
		UserID:          &userID,
		SessionID:       &session.SessionId,
		EventType:       models.SecurityEventLogin,
		EventCategory:   utils.PtrString("authentication"),
		Severity:        models.SecuritySeverityMedium,
		Status:          utils.PtrString("initiated"),
		Description:     utils.PtrString("User login attempt initiated"),
		IPAddress:       &locationInfo.IP,
		UserAgent:       &userAgent,
		DeviceID:        &device.ID,
		LocationCountry: &locationInfo.Country,
		LocationCity:    &locationInfo.City,
		IsSuspicious:    false,
		Metadata:        nil,
	})
	if err != nil {
		h.log.Error("Failed to log security event",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.String("session_id", session.SessionId),
			logger.Error(err),
		)
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	requestID := request.GetRequestID(r)
	correlationID := request.GetCorrelationID(r)

	h.log.Info("Login request received",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
		logger.String("client_ip", request.GetClientIP(r)),
	)

	handler := request.NewHandler(r, w)
	loginRequest := dto.NewLoginRequest()
	if !handler.ParseValidateAndSend(loginRequest) {
		h.log.Warn("Login request validation failed",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
		)
		return
	}

	deviceInfo := request.GetDeviceInfo(r)
	browserInfo := request.GetBrowserInfo(r)
	userAgent := request.GetUserAgent(r)
	clientIP := request.GetClientIP(r)

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

	user, authErr := h.service.GetUserByEmail(r.Context(), loginRequest.Email)
	if authErr != nil {
		h.log.Error("Failed to fetch user during login",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", loginRequest.Email),
			logger.String("error_code", authErr.Code),
			logger.Error(authErr.Error),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to process login", authErr.Error)
		return
	}
	if user == nil {
		h.log.Warn("Login attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("request_id", requestID),
			logger.String("email", loginRequest.Email),
		)
		response.BadRequestError(r.Context(), r, w, fmt.Sprintf("User does not exist with email %s", loginRequest.Email), nil)
		return
	}

	userResult, authErr := h.service.Login(r.Context(), loginRequest.Email, loginRequest.Password)
	if authErr != nil {
		if authErr.Code == authErrors.CodeInvalidCredentials {
			h.LogFailedLogin(r.Context(), deviceInfo, locationInfo, user.ID, userAgent, authErr.Message)
			response.BadRequestError(r.Context(), r, w, authErr.Message, nil)
		} else {
			response.InternalServerError(r.Context(), r, w, authErr.Message, authErr.Error)
		}
		return
	}
	if userResult == nil {
		response.BadRequestError(r.Context(), r, w, fmt.Sprintf("No user found with email %s", loginRequest.Email), nil)
		return
	}

	session := &serviceModels.CreateSessionOutput{}
	activeSession, sessErr := h.sessionService.GetSessionByUserId(r.Context(), user.ID)
	if sessErr != nil {
		h.log.Error("Failed to fetch active session during login", logger.Error(sessErr))
		response.InternalServerError(r.Context(), r, w, "Failed to process login", sessErr)
		return
	}

	if activeSession == nil {
		isMobile := deviceInfo.IsMobile()
		session, err = h.sessionService.CreateSession(r.Context(), serviceModels.CreateSessionInput{
			UserID:          userResult.User.ID,
			RefreshToken:    userResult.Session.RefreshToken,
			Device:          deviceInfo,
			Browser:         browserInfo,
			UserAgent:       request.GetUserAgent(r),
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
			Metadata: map[string]interface{}{
				"request_id":     request.GetRequestID(r),
				"correlation_id": request.GetCorrelationID(r),
			},
		})
		if err != nil {
			h.log.Error("Failed to create session after login", logger.Error(err))
			response.InternalServerError(r.Context(), r, w, "Failed to create session", err)
			return
		}
	} else {
		session.SessionId = activeSession.ID
		session.SessionToken = activeSession.SessionToken
		session.DeviceFingerprint = h.sessionService.GenerateDeviceFingerprint(*activeSession.DeviceID, *activeSession.DeviceOS, *activeSession.DeviceName)
	}

	h.LogSuccessfulLogin(r.Context(), session, deviceInfo, locationInfo, userResult.User.ID, userAgent, "User logged in successfully")

	h.log.Info("Login successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("request_id", requestID),
		logger.String("user_id", userResult.User.ID),
		logger.String("session_id", session.SessionId),
	)

	response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Login successful",
		map[string]any{
			"user":          userResult.User,
			"access_token":  userResult.Session.AccessToken,
			"expires_at":    userResult.Session.ExpiresAt,
			"refresh_token": userResult.Session.RefreshToken,
			"session_token": session.SessionToken,
		},
	)
}
