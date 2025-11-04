package service

import (
	"auth-service/internal/config"
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
		return "", fmt.Errorf("generate session nonce: %w", err)
	}

	payload := append(append([]byte{}, nonce...), []byte(userID)...)
	digest := sha256.Sum256(payload)

	tokenBytes := append(append([]byte{}, nonce...), digest[:]...)
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func (s *SessionService) GenerateDeviceFingerprint(deviceID string, deviceOS string, deviceName string) string {
	data := fmt.Sprintf("%s|%s|%s", deviceID, deviceOS, deviceName)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *SessionService) CreateSession(ctx context.Context, input serviceModels.CreateSessionInput) (*serviceModels.CreateSessionOutput, error) {
	sessionToken, err := s.generateSessionToken(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	err = s.repo.CreateSession(context.Background(), &models.AuthSession{
		ID:                 uuid.NewString(),
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
		PushEnabled:        input.FCMToken != "" || input.APNSToken != "",
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Cache the session token
	key := fmt.Sprintf("session_token:%s", sessionToken)
	value := []byte(input.UserID)
	err = s.cache.Set(ctx, key, value, 24*60*60)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	deviceFingerprint := s.GenerateDeviceFingerprint(input.Device.ID, input.Device.OS, input.Device.Name)
	return &serviceModels.CreateSessionOutput{
		SessionToken:      sessionToken,
		DeviceFingerprint: deviceFingerprint,
	}, nil
}

func (s *SessionService) GetSessionByUserId(ctx context.Context, userID string) (*models.AuthSession, error) {
	s.log.Debug("Fetching session by user ID",
		logger.String("user_id", userID),
	)
	session, err := s.repo.GetSessionByUserId(ctx, userID)
	if err != nil && !database.IsNoRowsError(err) {
		s.log.Error("Failed to get session by user ID", logger.Error(err))
		return nil, err
	}
	return session, nil
}

func (s *SessionService) DeleteSessionByID(ctx context.Context, sessionID string) error {
	s.log.Debug("Deleting session by ID",
		logger.String("session_id", sessionID),
	)
	err := s.repo.DeleteSessionByID(ctx, sessionID)
	if err != nil {
		s.log.Error("Failed to delete session by ID", logger.Error(err))
		return err
	}
	return nil
}
