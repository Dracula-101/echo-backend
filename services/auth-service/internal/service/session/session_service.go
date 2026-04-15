package session

import (
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"

	"github.com/google/uuid"
)

// SessionServiceInterface defines the contract for session service operations
type SessionService interface {
	// Session management
	PrepareSessionSlot(ctx context.Context, userID, deviceID string) pkgErrors.AppError
	CreateSession(ctx context.Context, input domain.CreateSessionInput) (*domain.CreateSessionOutput, pkgErrors.AppError)
	UpdateSession(ctx context.Context, userID string, sessionID string, updates domain.UpdateSession) (*domain.Session, pkgErrors.AppError)
	CheckAndHandleRefreshTokenReplay(ctx context.Context, refreshToken string) (bool, *authErrors.AuthError)
	GetSessionByUserId(ctx context.Context, userID string, deviceID string) (*domain.SessionRecord, pkgErrors.AppError)
	GetAllSessions(ctx context.Context, userID string, limit int, offset int) ([]*domain.Session, int, int, int, pkgErrors.AppError)
	GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError)
	DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError
}

func (s *sessionService) generateSessionToken(userID string) (string, pkgErrors.AppError) {
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

const maxActiveSessions = 10

func (s *sessionService) CreateSession(ctx context.Context, input domain.CreateSessionInput) (*domain.CreateSessionOutput, pkgErrors.AppError) {
	s.log.Info("Creating session",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", input.UserID),
		logger.String("device_os", input.Device.OS),
		logger.String("ip_address", input.IP.IP),
		logger.Bool("is_mobile", input.IsMobile),
	)

	sessionToken, err := s.generateSessionToken(input.UserID)
	if err != nil {
		return nil, pkgErrors.FromError(err, authErrors.CodeSessionCreationFailed, "failed to generate session token").
			WithService(authErrors.ServiceName).
			WithDetail("user_id", input.UserID)
	}

	sessionID := uuid.NewString()
	pushEnabled := input.FCMToken != "" || input.APNSToken != ""

	// Marshal metadata map into *json.RawMessage expected by models.AuthSession.Metadata
	var metadata *json.RawMessage
	if len(input.Metadata) > 0 {
		b, err := json.Marshal(input.Metadata)
		if err != nil {
			if appErr, ok := err.(pkgErrors.AppError); ok {
				return nil, appErr.WithService(authErrors.ServiceName)
			}
			return nil, pkgErrors.FromError(err, authErrors.CodeSessionCreationFailed, "failed to marshal session metadata").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", input.UserID)
		}
		raw := json.RawMessage(b)
		metadata = &raw
	}

	s.log.Debug("Storing session in database",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
		logger.String("user_id", input.UserID),
		logger.Bool("push_enabled", pushEnabled),
	)
	metaDataInfo := make(map[string]interface{})
	utils.UnmarshalRawMessageSafe(*metadata, metaDataInfo)
	err = s.repo.CreateSession(ctx, &domain.Session{
		ID:                 sessionID,
		UserID:             input.UserID,
		SessionToken:       sessionToken,
		RefreshToken:       input.RefreshToken,
		DeviceID:           input.Device.ID,
		DeviceName:         input.Device.Name,
		DeviceType:         input.Device.Type,
		DeviceOS:           input.Device.OS,
		DeviceOSVersion:    input.Device.OsVersion,
		DeviceModel:        input.Device.Model,
		DeviceManufacturer: input.Device.Manufacturer,
		BrowserName:        input.Browser.Name,
		BrowserVersion:     input.Browser.Version,
		UserAgent:          input.UserAgent,
		IPAddress:          input.IP.IP,
		Country:            input.IP.Country,
		Region:             input.IP.State,
		City:               input.IP.City,
		Timezone:           input.IP.Timezone,
		IPISP:              input.IP.ISP,
		Latitude:           &input.Latitude,
		Longitude:          &input.Longitude,
		IsMobile:           input.IsMobile,
		ExpiresAt:          input.ExpiresAt,
		IsTrustedDevice:    input.IsTrustedDevice,
		FCMToken:           input.FCMToken,
		APNSToken:          input.APNSToken,
		SessionType:        toDBSessionType(input.SessionType),
		PushEnabled:        pushEnabled,
		Metadata:           metaDataInfo,
	})
	if err != nil {
		return nil, pkgErrors.FromError(err, authErrors.CodeSessionCreationFailed, "failed to store session in database").
			WithService(authErrors.ServiceName).
			WithDetail("session_id", sessionID).
			WithDetail("user_id", input.UserID)
	}

	if cacheErr := s.sessionCache.SetSession(ctx, input.UserID, sessionID, sessionToken, input.Device.ID, input.ExpiresAt); cacheErr != nil {
		s.log.Warn("Failed to cache session (non-critical)",
			logger.String("service", authErrors.ServiceName),
			logger.String("session_id", sessionID),
			logger.String("user_id", input.UserID),
			logger.Error(cacheErr),
		)
	}

	s.log.Info("Session created successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
		logger.String("user_id", input.UserID),
	)

	return &domain.CreateSessionOutput{
		SessionId:    sessionID,
		SessionToken: sessionToken,
	}, nil
}

// PrepareSessionSlot handles same-device deduplication and session cap enforcement
// before a new session is created. Call this before CreateSession.
func (s *sessionService) PrepareSessionSlot(ctx context.Context, userID, deviceID string) pkgErrors.AppError {
	if deviceID != "" {
		revokedID, err := s.repo.RevokeActiveDeviceSession(ctx, userID, deviceID, "new_login_same_device")
		if err != nil {
			return pkgErrors.FromError(err, authErrors.CodeSessionCreationFailed, "failed to deduplicate device session").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID).
				WithDetail("device_id", deviceID)
		}
		if revokedID != "" {
			s.log.Info("Existing session on same device revoked",
				logger.String("service", authErrors.ServiceName),
				logger.String("user_id", userID),
				logger.String("device_id", deviceID),
				logger.String("revoked_session_id", revokedID),
			)
			if cacheErr := s.sessionCache.DeleteSession(ctx, revokedID); cacheErr != nil {
				s.log.Warn("Failed to delete replaced device session from cache",
					logger.String("service", authErrors.ServiceName),
					logger.String("session_id", revokedID),
					logger.Error(cacheErr),
				)
			}
		}
	}

	count, err := s.repo.CountActiveSessions(ctx, userID)
	if err != nil {
		s.log.Error("Failed to count active sessions",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return err
	}
	if count < maxActiveSessions {
		return nil
	}

	evicted, err := s.repo.EvictOldestSession(ctx, userID)
	if err != nil {
		s.log.Error("Failed to evict oldest session",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return err
	}
	if evicted == nil {
		return nil
	}

	s.log.Info("Session limit reached — oldest session evicted",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("evicted_session_id", evicted.ID),
		logger.Int64("active_count", count),
	)

	if cacheErr := s.sessionCache.DeleteSession(ctx, evicted.ID); cacheErr != nil {
		s.log.Warn("Failed to delete evicted session from cache",
			logger.String("service", authErrors.ServiceName),
			logger.String("session_id", evicted.ID),
			logger.Error(cacheErr),
		)
	}
	// TODO: emit outbox event for session revocation so ws-service can notify the device
	//       payload: { scope: "single", session_ids: [evicted.ID], reason: "session_limit_exceeded" }

	return nil
}

func (s *sessionService) UpdateSession(ctx context.Context, userID string, sessionID string, updates domain.UpdateSession) (*domain.Session, pkgErrors.AppError) {
	s.log.Info("Updating session",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	var update domain.UpdateAuthSession
	update.ID = sessionID
	if updates.FCMToken != nil {
		update.FCMToken = updates.FCMToken
	}
	if updates.APNSToken != nil {
		update.APNSToken = updates.APNSToken
	}
	if updates.PushEnabled != nil {
		update.PushEnabled = updates.PushEnabled
	}

	s.log.Debug("Saving updated session to database",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)
	// Save updated session
	session, dbErr := s.repo.UpdateSession(ctx, &update)
	if dbErr != nil {
		return nil, pkgErrors.FromError(dbErr, authErrors.CodeSessionUpdateFailed, "failed to update session in database").
			WithService(authErrors.ServiceName).
			WithDetail("session_id", sessionID)
	}

	s.log.Info("Session updated successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	return session, nil
}

func (s *sessionService) CheckAndHandleRefreshTokenReplay(ctx context.Context, refreshToken string) (bool, *authErrors.AuthError) {
	used, cacheErr := s.sessionCache.GetRefreshUsed(refreshToken)
	if cacheErr != nil {
		s.log.Error("Failed to check used refresh token in cache",
			logger.String("service", authErrors.ServiceName),
			logger.String("refresh_token", refreshToken),
			logger.Error(cacheErr),
		)
		return false, &authErrors.AuthError{
			Message: "Failed to check used refresh token in cache",
			Code:    authErrors.CodeCacheError,
			Error: pkgErrors.FromError(cacheErr, authErrors.CodeCacheError, "failed to check used refresh token in cache").
				WithService(authErrors.ServiceName).
				WithDetail("refresh_token", refreshToken),
		}
	}
	if used {
		s.log.Warn("Refresh token replay detected",
			logger.String("service", authErrors.ServiceName),
			logger.String("refresh_token", refreshToken),
		)
		// revoke the session associated with this refresh token
		sessionErr := s.repo.RevokeSession(ctx, refreshToken, "refresh_token_replay")
		if sessionErr != nil {
			s.log.Error("Failed to revoke session for refresh token replay",
				logger.String("service", authErrors.ServiceName),
				logger.String("refresh_token", refreshToken),
				logger.Error(sessionErr),
			)
			return false, &authErrors.AuthError{
				Message: "Failed to revoke session for refresh token replay",
				Code:    authErrors.CodeSessionRevocationFailed,
				Error: pkgErrors.FromError(sessionErr, authErrors.CodeSessionRevocationFailed, "failed to revoke session for refresh token replay").
					WithService(authErrors.ServiceName).
					WithDetail("refresh_token", refreshToken),
			}
		}
		return true, nil
	}
	return false, nil
}

func (s *sessionService) GetSessionByUserId(ctx context.Context, userID string, deviceID string) (*domain.SessionRecord, pkgErrors.AppError) {
	s.log.Debug("Fetching session by user ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	session, err := s.repo.GetSessionByUserId(ctx, userID, &deviceID)
	if err != nil {
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

	return &domain.SessionRecord{
		ID:           session.ID,
		SessionToken: session.SessionToken,
		RefreshToken: session.RefreshToken,
	}, nil
}

func (s *sessionService) DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError {
	s.log.Info("Deleting session",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	err := s.repo.DeleteSessionByID(ctx, sessionID)
	if err != nil {
		return err.WithService(authErrors.ServiceName)
	}

	s.sessionCache.DeleteSession(ctx, sessionID)

	s.log.Info("Session deleted successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	return nil
}

func (s *sessionService) GetAllSessions(ctx context.Context, userID string, limit int, offset int) ([]*domain.Session, int, int, int, pkgErrors.AppError) {
	s.log.Debug("Fetching all sessions for user",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	sessions, total, err := s.repo.GetAllSessionsByUserId(ctx, userID, limit, offset)
	if err != nil {
		return nil, limit, offset, total, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get all sessions").
			WithService(authErrors.ServiceName).
			WithDetail("user_id", userID)
	}
	return sessions, limit, offset, total, nil
}

func (s *sessionService) GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError) {
	s.log.Debug("Fetching session by ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("session_id", sessionID),
	)

	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get session by ID").
			WithService(authErrors.ServiceName).
			WithDetail("session_id", sessionID)
	}
	return session, nil
}
