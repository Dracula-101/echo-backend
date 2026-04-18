package ratelimiter

import (
	"context"
	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/windowstore"
	"time"
)

type RateLimiter interface {
	AllowLoginIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementLoginIP(ctx context.Context, ip string) (int64, pkgErrors.AppError)
	AllowLoginUser(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementLoginUser(ctx context.Context, userID string) (int64, pkgErrors.AppError)

	AllowRegisterIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementRegisterIP(ctx context.Context, ip string) (int64, pkgErrors.AppError)

	AllowPwdForgotIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementPwdForgotIP(ctx context.Context, ip string) (int64, pkgErrors.AppError)
	AllowPwdForgotEmail(ctx context.Context, email string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementPwdForgotEmail(ctx context.Context, email string) (int64, pkgErrors.AppError)

	AllowPwdResetIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementPwdResetIP(ctx context.Context, ip string) (int64, pkgErrors.AppError)
	SetPwdResetCooldown(ctx context.Context, userID string) pkgErrors.AppError
	CanSendPwdReset(ctx context.Context, userID string) (bool, int64, pkgErrors.AppError)

	AllowResendVerify(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementResendVerify(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowOtpSend(ctx context.Context, phone string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementOtpSend(ctx context.Context, phone string) (int64, pkgErrors.AppError)
	AllowRefreshDevice(ctx context.Context, deviceID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementRefreshDevice(ctx context.Context, deviceID string) (int64, pkgErrors.AppError)
	Allow2FA(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	Increment2FA(ctx context.Context, userID string) (int64, pkgErrors.AppError)

	AcquireLoginLock(ctx context.Context, userID string) (bool, pkgErrors.AppError)
	ReleaseLoginLock(ctx context.Context, userID string) pkgErrors.AppError
	AcquireRegisterLock(ctx context.Context, email string) (bool, pkgErrors.AppError)
	ReleaseRegisterLock(ctx context.Context, email string) pkgErrors.AppError

	IsIPBlocked(ctx context.Context, ip string) (bool, pkgErrors.AppError)
	BlockIP(ctx context.Context, ip string) pkgErrors.AppError
	IsAccountLocked(ctx context.Context, userID string) (bool, pkgErrors.AppError)
	LockAccount(ctx context.Context, userID string) pkgErrors.AppError
	LockAccountExtended(ctx context.Context, userID string) pkgErrors.AppError

	AllowProfileUpdate(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementProfileUpdate(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowContactAdd(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementContactAdd(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowBlockAction(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementBlockAction(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowSearch(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementSearch(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowReport(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementReport(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowStatusCreate(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementStatusCreate(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	AllowAvatarUpload(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError)
	IncrementAvatarUpload(ctx context.Context, userID string) (int64, pkgErrors.AppError)

	MarkStatusViewed(ctx context.Context, viewerID, statusID string) (bool, pkgErrors.AppError)

	AcquireContactLock(ctx context.Context, userID, targetID string) (bool, pkgErrors.AppError)
	ReleaseContactLock(ctx context.Context, userID, targetID string) pkgErrors.AppError
	AcquireBlockLock(ctx context.Context, userID, targetID string) (bool, pkgErrors.AppError)
	ReleaseBlockLock(ctx context.Context, userID, targetID string) pkgErrors.AppError

	SetHeartbeat(ctx context.Context, userID, deviceID string) pkgErrors.AppError
	GetHeartbeat(ctx context.Context, userID, deviceID string) (int64, pkgErrors.AppError)
	SetDeviceCount(ctx context.Context, userID string, count int64) pkgErrors.AppError
	GetDeviceCount(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	IncrementDeviceCount(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	DecrementDeviceCount(ctx context.Context, userID string) (int64, pkgErrors.AppError)

	GetPrivacyCache(ctx context.Context, userID, viewerID string) (string, bool, pkgErrors.AppError)
	SetPrivacyCache(ctx context.Context, userID, viewerID string, json string) pkgErrors.AppError
	DeletePrivacyCache(ctx context.Context, userID, viewerID string) pkgErrors.AppError
}

type rateLimiter struct {
	cache       cache.Cache
	windowStore windowstore.WindowStore
	log         logger.Logger
}

func NewRateLimiter(c cache.Cache, ws windowstore.WindowStore, log logger.Logger) RateLimiter {
	return &rateLimiter{cache: c, windowStore: ws, log: log}
}

// ── internal helpers ──────────────────────────────────────────────────────────

func (r *rateLimiter) allowWindow(ctx context.Context, key string, window time.Duration, limit int64) (bool, int64, pkgErrors.AppError) {
	count, err := r.incrementWindow(ctx, key, window)
	if err != nil {
		return false, 0, err
	}
	return count <= limit, count, nil
}

func (r *rateLimiter) incrementWindow(ctx context.Context, key string, window time.Duration) (int64, pkgErrors.AppError) {
	now := time.Now().Unix()
	cutoff := now - int64(window.Seconds())

	if err := r.windowStore.Add(ctx, key, now, window); err != nil {
		return 0, err
	}
	_ = r.windowStore.TrimBefore(ctx, key, cutoff)

	return r.windowStore.Count(ctx, key, cutoff, now)
}

// ── login ─────────────────────────────────────────────────────────────────────

func (r *rateLimiter) AllowLoginIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLLoginIPPrefix+ip, TTLLoginWindow, limit)
}

func (r *rateLimiter) IncrementLoginIP(ctx context.Context, ip string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLLoginIPPrefix+ip, TTLLoginWindow)
}

func (r *rateLimiter) AllowLoginUser(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLLoginUserPrefix+userID, TTLLoginWindow, limit)
}

func (r *rateLimiter) IncrementLoginUser(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLLoginUserPrefix+userID, TTLLoginWindow)
}

// ── register ──────────────────────────────────────────────────────────────────

func (r *rateLimiter) AllowRegisterIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLRegisterIPPrefix+ip, TTLRegisterIP, limit)
}

func (r *rateLimiter) IncrementRegisterIP(ctx context.Context, ip string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLRegisterIPPrefix+ip, TTLRegisterIP)
}

// ── forgot password ───────────────────────────────────────────────────────────

func (r *rateLimiter) AllowPwdForgotIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLPwdForgotIPPrefix+ip, TTLPwdForgotIP, limit)
}

func (r *rateLimiter) IncrementPwdForgotIP(ctx context.Context, ip string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLPwdForgotIPPrefix+ip, TTLPwdForgotIP)
}

func (r *rateLimiter) AllowPwdForgotEmail(ctx context.Context, email string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLPwdForgotEmailPrefix+email, TTLPwdForgotEmail, limit)
}

func (r *rateLimiter) IncrementPwdForgotEmail(ctx context.Context, email string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLPwdForgotEmailPrefix+email, TTLPwdForgotEmail)
}

// ── reset password ────────────────────────────────────────────────────────────

func (r *rateLimiter) AllowPwdResetIP(ctx context.Context, ip string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLPwdResetIPPrefix+ip, TTLPwdResetIP, limit)
}

func (r *rateLimiter) IncrementPwdResetIP(ctx context.Context, ip string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLPwdResetIPPrefix+ip, TTLPwdResetIP)
}

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

	return false, int64(TTLPwdResetSent.Seconds()) - elapsed, nil
}

// ── verify / otp / refresh / 2fa ─────────────────────────────────────────────

func (r *rateLimiter) AllowResendVerify(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLResendVerifyPrefix+userID, TTLResendVerify, limit)
}

func (r *rateLimiter) IncrementResendVerify(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLResendVerifyPrefix+userID, TTLResendVerify)
}

func (r *rateLimiter) AllowOtpSend(ctx context.Context, phone string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLOtpSendPhonePrefix+phone, TTLOtpSendPhone, limit)
}

func (r *rateLimiter) IncrementOtpSend(ctx context.Context, phone string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLOtpSendPhonePrefix+phone, TTLOtpSendPhone)
}

func (r *rateLimiter) AllowRefreshDevice(ctx context.Context, deviceID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLRefreshDevicePrefix+deviceID, TTLRefreshDevice, limit)
}

func (r *rateLimiter) IncrementRefreshDevice(ctx context.Context, deviceID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLRefreshDevicePrefix+deviceID, TTLRefreshDevice)
}

func (r *rateLimiter) Allow2FA(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RL2FAUserPrefix+userID, TTL2FAUser, limit)
}

func (r *rateLimiter) Increment2FA(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RL2FAUserPrefix+userID, TTL2FAUser)
}

// ── auth locks ────────────────────────────────────────────────────────────────

func (r *rateLimiter) AcquireLoginLock(ctx context.Context, userID string) (bool, pkgErrors.AppError) {
	ok, err := r.cache.AcquireLock(ctx, LockLoginPrefix+userID, TTLLockLogin)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire login lock")
	}
	return ok, nil
}

func (r *rateLimiter) ReleaseLoginLock(ctx context.Context, userID string) pkgErrors.AppError {
	if err := r.cache.ReleaseLock(ctx, LockLoginPrefix+userID); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to release login lock")
	}
	return nil
}

func (r *rateLimiter) AcquireRegisterLock(ctx context.Context, email string) (bool, pkgErrors.AppError) {
	ok, err := r.cache.AcquireLock(ctx, LockRegisterPrefix+email, TTLLockRegister)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire register lock")
	}
	return ok, nil
}

func (r *rateLimiter) ReleaseRegisterLock(ctx context.Context, email string) pkgErrors.AppError {
	if err := r.cache.ReleaseLock(ctx, LockRegisterPrefix+email); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to release register lock")
	}
	return nil
}

// ── ip / account protection ───────────────────────────────────────────────────

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

func (r *rateLimiter) LockAccountExtended(ctx context.Context, userID string) pkgErrors.AppError {
	return r.cache.SetBool(ctx, AccountLockedPrefix+userID, true, TTLAccountLockedExtended)
}

// ── social: window-based ──────────────────────────────────────────────────────

func (r *rateLimiter) AllowProfileUpdate(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLProfileUpdatePrefix+userID, TTLProfileUpdate, limit)
}

func (r *rateLimiter) IncrementProfileUpdate(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLProfileUpdatePrefix+userID, TTLProfileUpdate)
}

func (r *rateLimiter) AllowContactAdd(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLContactAddPrefix+userID, TTLContactAdd, limit)
}

func (r *rateLimiter) IncrementContactAdd(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLContactAddPrefix+userID, TTLContactAdd)
}

func (r *rateLimiter) AllowBlockAction(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLBlockPrefix+userID, TTLBlockAction, limit)
}

func (r *rateLimiter) IncrementBlockAction(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLBlockPrefix+userID, TTLBlockAction)
}

func (r *rateLimiter) AllowSearch(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLSearchPrefix+userID, TTLSearch, limit)
}

func (r *rateLimiter) IncrementSearch(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLSearchPrefix+userID, TTLSearch)
}

func (r *rateLimiter) AllowReport(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLReportPrefix+userID, TTLReport, limit)
}

func (r *rateLimiter) IncrementReport(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLReportPrefix+userID, TTLReport)
}

func (r *rateLimiter) AllowStatusCreate(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLStatusCreatePrefix+userID, TTLStatusCreate, limit)
}

func (r *rateLimiter) IncrementStatusCreate(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLStatusCreatePrefix+userID, TTLStatusCreate)
}

func (r *rateLimiter) AllowAvatarUpload(ctx context.Context, userID string, limit int64) (bool, int64, pkgErrors.AppError) {
	return r.allowWindow(ctx, RLAvatarUploadPrefix+userID, TTLAvatarUpload, limit)
}

func (r *rateLimiter) IncrementAvatarUpload(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	return r.incrementWindow(ctx, RLAvatarUploadPrefix+userID, TTLAvatarUpload)
}

// ── status view dedup ─────────────────────────────────────────────────────────

func (r *rateLimiter) MarkStatusViewed(ctx context.Context, viewerID, statusID string) (bool, pkgErrors.AppError) {
	key := RLStatusViewPrefix + viewerID + ":" + statusID

	exists, err := r.cache.Exists(ctx, key)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check status view key")
	}
	if exists {
		return false, nil
	}

	if err := r.cache.SetBool(ctx, key, true, TTLStatusView); err != nil {
		return false, err
	}
	return true, nil
}

// ── social locks ──────────────────────────────────────────────────────────────

func (r *rateLimiter) AcquireContactLock(ctx context.Context, userID, targetID string) (bool, pkgErrors.AppError) {
	ok, err := r.cache.AcquireLock(ctx, LockContactPrefix+userID+":"+targetID, TTLLockContact)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire contact lock")
	}
	return ok, nil
}

func (r *rateLimiter) ReleaseContactLock(ctx context.Context, userID, targetID string) pkgErrors.AppError {
	if err := r.cache.ReleaseLock(ctx, LockContactPrefix+userID+":"+targetID); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to release contact lock")
	}
	return nil
}

func (r *rateLimiter) AcquireBlockLock(ctx context.Context, userID, targetID string) (bool, pkgErrors.AppError) {
	ok, err := r.cache.AcquireLock(ctx, LockBlockPrefix+userID+":"+targetID, TTLLockBlock)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire block lock")
	}
	return ok, nil
}

func (r *rateLimiter) ReleaseBlockLock(ctx context.Context, userID, targetID string) pkgErrors.AppError {
	if err := r.cache.ReleaseLock(ctx, LockBlockPrefix+userID+":"+targetID); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to release block lock")
	}
	return nil
}

// ── presence ──────────────────────────────────────────────────────────────────

func (r *rateLimiter) SetHeartbeat(ctx context.Context, userID, deviceID string) pkgErrors.AppError {
	return r.cache.SetInt(ctx, HeartbeatPrefix+userID+":"+deviceID, time.Now().Unix(), TTLHeartbeat)
}

func (r *rateLimiter) GetHeartbeat(ctx context.Context, userID, deviceID string) (int64, pkgErrors.AppError) {
	ts, err := r.cache.GetInt(ctx, HeartbeatPrefix+userID+":"+deviceID)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get heartbeat")
	}
	return ts, nil
}

func (r *rateLimiter) SetDeviceCount(ctx context.Context, userID string, count int64) pkgErrors.AppError {
	return r.cache.SetInt(ctx, DeviceCountPrefix+userID, count, TTLDeviceCount)
}

func (r *rateLimiter) GetDeviceCount(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	count, err := r.cache.GetInt(ctx, DeviceCountPrefix+userID)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get device count")
	}
	return count, nil
}

func (r *rateLimiter) IncrementDeviceCount(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	count, err := r.cache.Increment(ctx, DeviceCountPrefix+userID, 1)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to increment device count")
	}
	_ = r.cache.Expire(ctx, DeviceCountPrefix+userID, TTLDeviceCount)
	return count, nil
}

func (r *rateLimiter) DecrementDeviceCount(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	count, err := r.cache.Decrement(ctx, DeviceCountPrefix+userID, 1)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to decrement device count")
	}
	_ = r.cache.Expire(ctx, DeviceCountPrefix+userID, TTLDeviceCount)
	return count, nil
}

// ── privacy cache ─────────────────────────────────────────────────────────────

func (r *rateLimiter) GetPrivacyCache(ctx context.Context, userID, viewerID string) (string, bool, pkgErrors.AppError) {
	key := PrivacyCachePrefix + userID + ":" + viewerID

	exists, err := r.cache.Exists(ctx, key)
	if err != nil {
		return "", false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to check privacy cache")
	}
	if !exists {
		return "", false, nil
	}

	val, appErr := r.cache.GetString(ctx, key)
	if appErr != nil {
		return "", false, appErr
	}
	return val, true, nil
}

func (r *rateLimiter) SetPrivacyCache(ctx context.Context, userID, viewerID string, json string) pkgErrors.AppError {
	return r.cache.SetString(ctx, PrivacyCachePrefix+userID+":"+viewerID, json, TTLPrivacyCache)
}

func (r *rateLimiter) DeletePrivacyCache(ctx context.Context, userID, viewerID string) pkgErrors.AppError {
	return r.cache.Delete(ctx, PrivacyCachePrefix+userID+":"+viewerID)
}
