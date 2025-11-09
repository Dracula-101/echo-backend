package service

import (
	"auth-service/internal/config"
	authErrors "auth-service/internal/errors"
	repository "auth-service/internal/repo"
	serviceModels "auth-service/internal/service/models"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/common/token"

	"github.com/google/uuid"
)

type SessionService struct {
	repo         *repository.SessionRepo
	tokenService token.JWTTokenService
	cfg          config.CacheConfig
	cache        cache.Cache
	log          logger.Logger
}

func NewSessionService(repo *repository.SessionRepo, cache cache.Cache, token token.JWTTokenService, log logger.Logger, cfg config.CacheConfig) *SessionService {
	if repo == nil {
		panic("SessionRepo is required")
	}
	if log == nil {
		panic("Logger is required")
	}

	log.Info("Initializing SessionService",
		logger.String("service", authErrors.ServiceName),
	)

	return &SessionService{
		repo:         repo,
		cache:        cache,
		tokenService: token,
		log:          log,
		cfg:          cfg,
	}
}

func (s *SessionService) generateSessionToken(userID string) (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to generate session nonce").
			WithDetail("user_id", userID)
	}

	payload := append(append([]byte{}, nonce...), []byte(userID)...)
	digest := sha256.Sum256(payload)

	tokenBytes := append(append([]byte{}, nonce...), digest[:]...)
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	s.log.Debug("Generated session token",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int("token_length", len(token)),
	)

	return token, nil
}

func (s *SessionService) GenerateDeviceFingerprint(deviceID string, deviceOS string, deviceName string) string {
	data := fmt.Sprintf("%s|%s|%s", deviceID, deviceOS, deviceName)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *SessionService) CreateSession(ctx context.Context, input serviceModels.CreateSessionInput) (*serviceModels.CreateSessionOutput, error) {
	s.log.Info("Creating session",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", input.UserID),
		logger.String("device_os", input.Device.OS),
		logger.String("ip_address", input.IP.IP),
		logger.Bool("is_mobile", input.IsMobile),
	)

	sessionToken, err := s.generateSessionToken(input.UserID)
	if err != nil {
		if appErr, ok := err.(pkgErrors.AppError); ok {
			return nil, appErr.WithService(authErrors.ServiceName)
		}
		return nil, pkgErrors.FromError(err, authErrors.CodeSessionCreationFailed, "failed to generate session token").
			WithService(authErrors.ServiceName).
			WithDetail("user_id", input.UserID)
	}

	sessionID := uuid.NewString()
	pushEnabled := input.FCMToken != "" || input.APNSToken != ""

	s.log.Debug("Storing session in database",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
		logger.String("user_id", input.UserID),
		logger.Bool("push_enabled", pushEnabled),
	)

	err = s.repo.CreateSession(ctx, &models.AuthSession{
		ID:                 sessionID,
		UserID:             input.UserID,
		SessionToken:       sessionToken,
		RefreshToken:       &input.RefreshToken,
		DeviceID:           &input.Device.ID,
		DeviceName:         &input.Device.Name,
		DeviceType:         &input.Device.Type,
		DeviceOS:           &input.Device.OS,
		DeviceOSVersion:    &input.Device.OsVersion,
		DeviceModel:        &input.Device.Model,
		DeviceManufacturer: &input.Device.Manufacturer,
		BrowserName:        &input.Browser.Name,
		BrowserVersion:     &input.Browser.Version,
		UserAgent:          &input.UserAgent,
		IPAddress:          input.IP.IP,
		IPCountry:          &input.IP.Country,
		IPRegion:           &input.IP.Region,
		IPCity:             &input.IP.City,
		IPTimezone:         &input.IP.Timezone,
		IPISP:              &input.IP.ISP,
		Latitude:           &input.Latitude,
		Longitude:          &input.Longitude,
		IsMobile:           input.IsMobile,
		IsTrustedDevice:    input.IsTrustedDevice,
		FCMToken:           &input.FCMToken,
		APNSToken:          &input.APNSToken,
		SessionType:        input.SessionType,
		PushEnabled:        pushEnabled,
	})
	if err != nil {
		if appErr, ok := err.(pkgErrors.AppError); ok {
			return nil, appErr.WithService(authErrors.ServiceName)
		}
		return nil, pkgErrors.FromError(err, authErrors.CodeSessionCreationFailed, "failed to store session in database").
			WithService(authErrors.ServiceName).
			WithDetail("session_id", sessionID).
			WithDetail("user_id", input.UserID)
	}

	if s.cache != nil {
		key := fmt.Sprintf("session_token:%s", sessionToken)
		value := []byte(input.UserID)
		err = s.cache.Set(ctx, key, value, 24*60*60)
		if err != nil {
			s.log.Warn("Failed to cache session token (non-critical)",
				logger.String("service", authErrors.ServiceName),
				logger.String("session_id", sessionID),
				logger.String("user_id", input.UserID),
				logger.Error(err),
			)
		} else {
			s.log.Debug("Session token cached",
				logger.String("service", authErrors.ServiceName),
				logger.String("session_id", sessionID),
				logger.String("cache_key", key),
			)
		}
	}

	deviceFingerprint := s.GenerateDeviceFingerprint(input.Device.ID, input.Device.OS, input.Device.Name)

	s.log.Info("Session created successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
		logger.String("user_id", input.UserID),
		logger.String("device_fingerprint", deviceFingerprint),
	)

	return &serviceModels.CreateSessionOutput{
		SessionId:         sessionID,
		SessionToken:      sessionToken,
		DeviceFingerprint: deviceFingerprint,
	}, nil
}

func (s *SessionService) GetSessionByUserId(ctx context.Context, userID string) (*models.AuthSession, error) {
	s.log.Debug("Fetching session by user ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	session, err := s.repo.GetSessionByUserId(ctx, userID)
	if err != nil && !database.IsNoRowsError(err) {
		if appErr, ok := err.(pkgErrors.AppError); ok {
			return nil, appErr.WithService(authErrors.ServiceName)
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get session by user ID").
			WithService(authErrors.ServiceName).
			WithDetail("user_id", userID)
	}

	if session == nil {
		s.log.Debug("No active session found for user",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
		)
		return nil, nil
	}

	s.log.Debug("Session found",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("session_id", session.ID),
	)

	return session, nil
}

func (s *SessionService) DeleteSessionByID(ctx context.Context, sessionID string) error {
	s.log.Info("Deleting session",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	err := s.repo.DeleteSessionByID(ctx, sessionID)
	if err != nil {
		return err.WithService(authErrors.ServiceName)
	}

	s.log.Info("Session deleted successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	return nil
}
