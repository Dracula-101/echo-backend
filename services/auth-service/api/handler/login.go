package handler

import (
	"auth-service/api/dto"
	repositoryModels "auth-service/internal/repo/models"
	"auth-service/internal/service"
	serviceModels "auth-service/internal/service/models"
	"context"
	"fmt"
	"net/http"
	"shared/pkg/database/postgres/models"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	"shared/server/response"
)

func (h *AuthHandler) LogFailedLogin(ctx context.Context, device request.DeviceInfo, locationInfo *request.IpAddressInfo, userID string, userAgent string, failureReason string) {
	h.service.LoginHistoryRepo.CreateLoginHistory(ctx, repositoryModels.CreateLoginHistoryInput{
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
	h.service.FailedLogin(ctx, userID)
	h.log.Info("Failed login attempt", logger.String("user_id", userID), logger.String("reason", failureReason))
}

func (h *AuthHandler) LogSuccessfulLogin(ctx context.Context, session *serviceModels.CreateSessionOutput, device request.DeviceInfo, locationInfo *request.IpAddressInfo, userID string, userAgent string, failureReason string) {
	h.service.SuccessLogin(ctx, userID)
	h.service.LoginHistoryRepo.CreateLoginHistory(ctx, repositoryModels.CreateLoginHistoryInput{
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
	h.service.SecurityEventRepo.LogSecurityEvent(ctx, &models.SecurityEvent{
		UserID:          &userID,
		SessionID:       &session.SessionId,
		EventType:       "login_attempt",
		EventCategory:   utils.PtrString("authentication"),
		Severity:        "medium",
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
	h.log.Info("Successful login recorded", logger.String("user_id", userID))
}

// ========================== Login Handler ==========================
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Validation and parsing
	handler := request.NewHandler(r, w)
	loginRequest := dto.NewLoginRequest()
	if !handler.ParseValidateAndSend(loginRequest) {
		return
	}
	h.log.Info("Login attempt received",
		logger.String("email", loginRequest.Email),
		logger.String("client_ip", request.GetClientIP(r)),
		logger.Any("device_info", request.GetDeviceInfo(r)),
	)

	// Extract request info
	deviceInfo := request.GetDeviceInfo(r)
	browserInfo := request.GetBrowserInfo(r)
	userAgent := request.GetUserAgent(r)
	clientIP := request.GetClientIP(r)

	// Lookup location
	h.log.Debug("Looking up location for IP", logger.String("ip", clientIP))
	locationInfo, err := h.locationService.Lookup(clientIP)
	if err != nil {
		h.log.Error("Failed to lookup location", logger.Error(err))
	}

	// Fetch user by email
	user, authErr := h.service.GetUserByEmail(r.Context(), loginRequest.Email)
	if authErr != nil {
		h.log.Error("Failed to fetch user during login", logger.Error(authErr.Error))
		response.InternalServerError(r.Context(), r, w, "Failed to process login", authErr.Error)
		return
	}
	if user == nil {
		response.BadRequestError(r.Context(), r, w, fmt.Sprintf("User does not exist with email %s", loginRequest.Email), nil)
		return
	}

	// Authenticate user
	userResult, authErr := h.service.Login(r.Context(), loginRequest.Email, loginRequest.Password)
	if authErr != nil {
		if authErr.Code == service.AuthErrorInvalidCredentials {
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

	// Create or fetch session
	session := &serviceModels.CreateSessionOutput{}
	activeSession, sessErr := h.sessionService.GetSessionByUserId(r.Context(), user.ID)
	if sessErr != nil {
		h.log.Error("Failed to fetch active session during login", logger.Error(sessErr))
		response.InternalServerError(r.Context(), r, w, "Failed to process login", sessErr)
		return
	}
	if activeSession != nil {
		h.log.Info("Active session found for user during login",
			logger.String("user_id", user.ID),
			logger.String("session_id", activeSession.ID),
		)
	}
	if activeSession == nil {
		session, err = h.sessionService.CreateSession(r.Context(), serviceModels.CreateSessionInput{
			UserID:          userResult.User.ID,
			RefreshToken:    userResult.Session.RefreshToken,
			Device:          deviceInfo,
			Browser:         browserInfo,
			UserAgent:       request.GetUserAgent(r),
			IP:              *locationInfo,
			Latitude:        locationInfo.Latitude,
			Longitude:       locationInfo.Longitude,
			IsMobile:        deviceInfo.OS == "iOS" || deviceInfo.OS == "Android",
			IsTrustedDevice: false,
			FCMToken:        loginRequest.FCMToken,
			APNSToken:       loginRequest.APNSToken,
			SessionType:     "login",
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

	// Log login attempt event
	h.LogSuccessfulLogin(r.Context(), session, deviceInfo, locationInfo, userResult.User.ID, userAgent, "User logged in successfully")
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
