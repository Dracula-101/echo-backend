package handler

import (
	"auth-service/api/v1/dto"
	authErrors "auth-service/internal/errors"
	repositoryModels "auth-service/internal/repo/models"
	serviceModels "auth-service/internal/service/models"
	"context"
	"errors"
	"fmt"
	"net/http"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
	"shared/server/response"
	"time"
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
		var appErr pkgErrors.AppError
		if errors.As(err, &appErr) {
			h.log.Error("Failed to create login history record",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.String("correlation_id", appErr.CorrelationID()),
				logger.Any("error_details", appErr.Details()),
				logger.Any("stack_trace", appErr.StackTrace()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to create login history record", logger.Error(err))
		}
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
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

	user, authErr := h.service.GetUserByEmail(r.Context(), loginRequest.Email)
	if authErr != nil {
		var appErr pkgErrors.AppError
		if errors.As(authErr, &appErr) {
			h.log.Error("Failed to fetch user during login",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.String("correlation_id", appErr.CorrelationID()),
				logger.Any("error_details", appErr.Details()),
				logger.Any("stack_trace", appErr.StackTrace()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to fetch user during login", logger.Error(authErr))
		}
		response.InternalServerError(r.Context(), r, w, "Failed to process login", authErr)
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
		var appErr pkgErrors.AppError
		if errors.As(authErr, &appErr) {
			h.log.Error("Login failed",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.String("correlation_id", appErr.CorrelationID()),
				logger.Any("error_details", appErr.Details()),
				logger.Any("stack_trace", appErr.StackTrace()),
				logger.Error(appErr),
			)
			if appErr.Code() == authErrors.CodeInvalidCredentials {
				h.LogFailedLogin(r.Context(), deviceInfo, locationInfo, user.ID, userAgent, appErr.Message())
				response.BadRequestError(r.Context(), r, w, appErr.Message(), nil)
			} else {
				response.InternalServerError(r.Context(), r, w, appErr.Message(), authErr)
			}
		} else {
			h.log.Error("Login failed", logger.Error(authErr))
			response.InternalServerError(r.Context(), r, w, "Failed to process login", authErr)
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
		var appErr pkgErrors.AppError
		if errors.As(sessErr, &appErr) {
			h.log.Error("Failed to fetch active session during login",
				logger.String("error_code", appErr.Code()),
				logger.String("service", appErr.Service()),
				logger.String("correlation_id", appErr.CorrelationID()),
				logger.Any("error_details", appErr.Details()),
				logger.Any("stack_trace", appErr.StackTrace()),
				logger.Error(appErr),
			)
		} else {
			h.log.Error("Failed to fetch active session during login", logger.Error(sessErr))
		}
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
			var appErr pkgErrors.AppError
			if errors.As(err, &appErr) {
				h.log.Error("Failed to create session after login",
					logger.String("error_code", appErr.Code()),
					logger.String("service", appErr.Service()),
					logger.String("correlation_id", appErr.CorrelationID()),
					logger.Any("error_details", appErr.Details()),
					logger.Any("stack_trace", appErr.StackTrace()),
					logger.Error(appErr),
				)
			} else {
				h.log.Error("Failed to create session after login", logger.Error(err))
			}
			response.InternalServerError(r.Context(), r, w, "Failed to create session", err)
			return
		}
	} else {
		session.SessionId = activeSession.ID
		session.SessionToken = activeSession.SessionToken
	}

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
			"session_id":    session.SessionId,
		},
	)
}
