package session

import (
	"context"
	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"time"
)

var (
	PreAuthTTL     = time.Second * 300   // 5 minutes
	Session2FA_TTL = time.Second * 600   // 10 minutes
	RefreshUsedTTL = time.Second * 13600 // 3 hours 46 minutes 40 seconds
	TOTPUsedTTL    = time.Second * 90    // 1.5 minutes
	OAuthJWKTTL    = time.Second * 3600  // 1 hour
	UserStatusTTL  = time.Second * 60    // 1 minute
)

type sessionCache struct {
	cache cache.Cache
	log   logger.Logger
}

func NewSessionCache(cache cache.Cache, log logger.Logger) SessionCache {
	if cache == nil {
		panic("Cache is required for SessionCache")
	}
	return &sessionCache{
		cache: cache,
		log:   log,
	}
}

type SessionCache interface {
	GetSession(sessionToken string) (*SessionData, pkgErrors.AppError)
	DeleteSession(ctx context.Context, sessionID string) pkgErrors.AppError
	SetSession(context context.Context, userID string, sessionId string, deviceID string, expiresAt time.Time, accountStatus string) pkgErrors.AppError
	SetPre2FA_Auth(context context.Context, token string, userID string, deviceInfo request.DeviceInfo, ip string) pkgErrors.AppError
	Set2FA_Setup(context context.Context, userID string, secretBase32 string, backupCodesPlainJSON string) pkgErrors.AppError
	SetRefreshUsed(context context.Context, refreshToken string) pkgErrors.AppError
	SetTOTPUsed(context context.Context, userID string, code string) pkgErrors.AppError
	SetOAuthJWK(context context.Context, provider string, jwkSetJSON string) pkgErrors.AppError
	SetUserStatus(context context.Context, userID string, accountStatus string, twoFactorEnabled bool) pkgErrors.AppError

	GetPre2FA_Auth(token string) (*Pre2FAData, pkgErrors.AppError)
	Get2FA_Setup(userID string) (*Setup2FAData, pkgErrors.AppError)
	GetRefreshUsed(refreshToken string) (bool, pkgErrors.AppError)
	GetTOTPUsed(userID string, code string) (bool, pkgErrors.AppError)
	GetOAuthJWK(provider string) (string, pkgErrors.AppError)
	GetUserStatus(userID string) (*UserStatusData, pkgErrors.AppError)
}

func (c *sessionCache) DeleteSession(ctx context.Context, sessionID string) pkgErrors.AppError {
	return c.cache.Delete(ctx, "session:"+sessionID)
}

func (c *sessionCache) SetSession(context context.Context, userID string, sessionId string, deviceID string, expiresAt time.Time, accountStatus string) pkgErrors.AppError {
	key := "session:" + sessionId
	value := SessionData{
		UserID:        userID,
		SessionID:     sessionId,
		DeviceID:      deviceID,
		ExpiresAt:     expiresAt.Unix(),
		AccountStatus: accountStatus,
	}
	return c.cache.Set(context, key, value.Bytes(), time.Until(expiresAt))
}

func (c *sessionCache) SetPre2FA_Auth(context context.Context, token string, userID string, deviceInfo request.DeviceInfo, ip string) pkgErrors.AppError {
	key := "pre_auth:" + token
	value := Pre2FAData{
		UserID:    userID,
		DeviceID:  deviceInfo.ID,
		IpAddress: ip,
		CreatedAt: time.Now().Unix(),
	}
	return c.cache.Set(context, key, value.Bytes(), PreAuthTTL)
}

func (c *sessionCache) Set2FA_Setup(context context.Context, userID string, secretBase32 string, backupCodesPlainJSON string) pkgErrors.AppError {
	key := "2fa_setup:" + userID
	value := Setup2FAData{
		UserID:       userID,
		SecretBase32: secretBase32,
		BackupCodes:  []string{backupCodesPlainJSON},
	}
	return c.cache.Set(context, key, value.Bytes(), Session2FA_TTL)
}

func (c *sessionCache) SetRefreshUsed(context context.Context, refreshToken string) pkgErrors.AppError {
	key := "refresh_used:" + refreshToken
	return c.cache.Set(context, key, []byte("1"), RefreshUsedTTL)
}

func (c *sessionCache) SetTOTPUsed(context context.Context, userID string, code string) pkgErrors.AppError {
	key := "totp_used:" + userID + ":" + code
	return c.cache.Set(context, key, []byte("1"), TOTPUsedTTL)
}

func (c *sessionCache) SetOAuthJWK(context context.Context, provider string, jwkSetJSON string) pkgErrors.AppError {
	key := "oauth_jwk:" + provider
	return c.cache.Set(context, key, []byte(jwkSetJSON), OAuthJWKTTL)
}

func (c *sessionCache) SetUserStatus(context context.Context, userID string, accountStatus string, twoFactorEnabled bool) pkgErrors.AppError {
	key := "user_status:" + userID
	value := UserStatusData{
		AccountStatus:    accountStatus,
		TwoFactorEnabled: twoFactorEnabled,
	}
	return c.cache.Set(context, key, value.Bytes(), UserStatusTTL)
}

func (c *sessionCache) GetSession(sessionToken string) (*SessionData, pkgErrors.AppError) {
	key := "session:" + sessionToken
	dataBytes, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get session from cache")
	}
	var data SessionData
	err = data.FromBytes(dataBytes)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to parse session data from cache")
	}
	return &data, nil
}

func (c *sessionCache) GetPre2FA_Auth(token string) (*Pre2FAData, pkgErrors.AppError) {
	key := "pre_auth:" + token
	dataBytes, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get pre-2FA auth data from cache")
	}
	var data Pre2FAData
	err = data.FromBytes(dataBytes)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to parse pre-2FA auth data from cache")
	}
	return &data, nil
}

func (c *sessionCache) Get2FA_Setup(userID string) (*Setup2FAData, pkgErrors.AppError) {
	key := "2fa_setup:" + userID
	dataBytes, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get 2FA setup data from cache")
	}
	var data Setup2FAData
	err = data.FromBytes(dataBytes)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to parse 2FA setup data from cache")
	}
	return &data, nil
}

func (c *sessionCache) GetRefreshUsed(refreshToken string) (bool, pkgErrors.AppError) {
	key := "refresh_used:" + refreshToken
	_, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return false, nil
		}
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check if refresh token was used")
	}
	return true, nil
}

func (c *sessionCache) GetTOTPUsed(userID string, code string) (bool, pkgErrors.AppError) {
	key := "totp_used:" + userID + ":" + code
	_, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return false, nil
		}
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check if TOTP code was used")
	}
	return true, nil
}

func (c *sessionCache) GetOAuthJWK(provider string) (string, pkgErrors.AppError) {
	key := "oauth_jwk:" + provider
	dataBytes, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return "", nil
		}
		return "", pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get OAuth JWK from cache")
	}
	return string(dataBytes), nil
}

func (c *sessionCache) GetUserStatus(userID string) (*UserStatusData, pkgErrors.AppError) {
	key := "user_status:" + userID
	dataBytes, err := c.cache.Get(context.Background(), key)
	if err != nil {
		if err == cache.ErrNotFound {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get user status from cache")
	}
	var data UserStatusData
	err = data.FromBytes(dataBytes)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to parse user status data from cache")
	}
	return &data, nil
}
