package ratelimiter

import (
	"context"
	"time"

	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/windowstore"
)

type RateLimiter interface {
	AllowLoginIP(ctx context.Context, ip string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowLoginUser(ctx context.Context, userID string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowRegisterIP(ctx context.Context, ip string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowPwdForgot(ctx context.Context, email string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowPwdResetIP(ctx context.Context, ip string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowResendVerify(ctx context.Context, userID string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowOtpSend(ctx context.Context, phone string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	AllowRefreshDevice(ctx context.Context, deviceID string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)
	Allow2FA(ctx context.Context, userID string, limit int64) (allowed bool, remaining int64, error pkgErrors.AppError)

	AcquireLoginLock(ctx context.Context, userID string) (allowed bool, error pkgErrors.AppError)
	AcquireRegisterLock(ctx context.Context, email string) (allowed bool, error pkgErrors.AppError)

	IsIPBlocked(ctx context.Context, ip string) (blocked bool, error pkgErrors.AppError)
	BlockIP(ctx context.Context, ip string) (error pkgErrors.AppError)

	IsAccountLocked(ctx context.Context, userID string) (locked bool, error pkgErrors.AppError)
	LockAccount(ctx context.Context, userID string) pkgErrors.AppError

	SetPwdResetCooldown(ctx context.Context, userID string) pkgErrors.AppError
	CanSendPwdReset(ctx context.Context, userID string) (canSend bool, retryAfterSeconds int64, error pkgErrors.AppError)
}

type rateLimiter struct {
	cache       cache.Cache
	windowStore windowstore.WindowStore
	log         logger.Logger
}

func NewRateLimiter(cache cache.Cache, ws windowstore.WindowStore, log logger.Logger) RateLimiter {
	return &rateLimiter{
		cache:       cache,
		windowStore: ws,
		log:         log,
	}
}

// ---------------- internal helper (windowstore based) ----------------

func (r *rateLimiter) allowWindow(
	ctx context.Context,
	key string,
	window time.Duration,
	limit int64,
) (allowed bool, remaining int64, error pkgErrors.AppError) {

	now := time.Now().Unix()

	err := r.windowStore.Add(ctx, key, now, window)
	if err != nil {
		return false, 0, err
	}

	_ = r.windowStore.TrimBefore(ctx, key, now-int64(window.Seconds()))

	count, err2 := r.windowStore.Count(ctx, key, now-int64(window.Seconds()), now)
	if err2 != nil {
		return false, 0, err2
	}

	if count > limit {
		return false, count, nil
	}

	return true, count, nil
}

// ---------------- rate limit funcs ----------------

func (r *rateLimiter) AllowLoginIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLLoginIPPrefix+ip, TTLLoginWindow, limit)
}

func (r *rateLimiter) AllowLoginUser(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLLoginUserPrefix+userID, TTLLoginWindow, limit)
}

func (r *rateLimiter) AllowRegisterIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLRegisterIPPrefix+ip, TTLRegisterIP, limit)
}

func (r *rateLimiter) AllowPwdForgot(ctx context.Context, email string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLPwdForgotEmailPrefix+email, TTLPwdForgotEmail, limit)
}

func (r *rateLimiter) AllowPwdResetIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLPwdResetIPPrefix+ip, TTLPwdResetIP, limit)
}

func (r *rateLimiter) AllowResendVerify(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLResendVerifyPrefix+userID, TTLResendVerify, limit)
}

func (r *rateLimiter) AllowOtpSend(ctx context.Context, phone string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLOtpSendPhonePrefix+phone, TTLOtpSendPhone, limit)
}

func (r *rateLimiter) AllowRefreshDevice(ctx context.Context, deviceID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLRefreshDevicePrefix+deviceID, TTLRefreshDevice, limit)
}

func (r *rateLimiter) Allow2FA(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RL2FAUserPrefix+userID, TTL2FAUser, limit)
}

// ---------------- locks ----------------

func (r *rateLimiter) AcquireLoginLock(ctx context.Context, userID string) (bool, pkgErrors.AppError) {
	ok, err := r.cache.AcquireLock(ctx, LockLoginPrefix+userID, TTLLockLogin)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire login lock")
	}
	return ok, nil
}

func (r *rateLimiter) AcquireRegisterLock(ctx context.Context, email string) (bool, pkgErrors.AppError) {
	ok, err := r.cache.AcquireLock(ctx, LockRegisterPrefix+email, TTLLockRegister)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire register lock")
	}
	return ok, nil
}

// ---------------- IP block ----------------

func (r *rateLimiter) IsIPBlocked(ctx context.Context, ip string) (bool, pkgErrors.AppError) {
	exists, err := r.cache.Exists(ctx, IPBlockedPrefix+ip)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check if IP is blocked")
	}
	return exists, nil
}

func (r *rateLimiter) BlockIP(ctx context.Context, ip string) pkgErrors.AppError {
	return r.cache.SetBool(ctx, IPBlockedPrefix+ip, true, TTLIPBlocked)
}

// ---------------- account lock ----------------

func (r *rateLimiter) IsAccountLocked(ctx context.Context, userID string) (bool, pkgErrors.AppError) {
	exists, err := r.cache.Exists(ctx, AccountLockedPrefix+userID)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check if account is locked")
	}
	return exists, nil
}

func (r *rateLimiter) LockAccount(ctx context.Context, userID string) pkgErrors.AppError {
	return r.cache.SetBool(ctx, AccountLockedPrefix+userID, true, TTLAccountLocked)
}

// ---------------- password reset ----------------

func (r *rateLimiter) SetPwdResetCooldown(ctx context.Context, userID string) pkgErrors.AppError {
	return r.cache.SetInt(ctx, PwdResetSentPrefix+userID, time.Now().Unix(), TTLPwdResetSent)
}

func (r *rateLimiter) CanSendPwdReset(ctx context.Context, userID string) (bool, int64, pkgErrors.AppError) {
	exists, err := r.cache.Exists(ctx, PwdResetSentPrefix+userID)
	if err != nil {
		return false, 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check password reset cooldown")
	}
	if !exists {
		return true, 0, nil
	}

	ts, err := r.cache.GetInt(ctx, PwdResetSentPrefix+userID)
	if err != nil {
		return false, 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get password reset timestamp")
	}

	elapsed := time.Now().Unix() - ts
	if elapsed >= int64(TTLPwdResetSent.Seconds()) {
		return true, 0, nil
	}

	retryAfter := int64(TTLPwdResetSent.Seconds()) - elapsed
	return false, retryAfter, nil
}
