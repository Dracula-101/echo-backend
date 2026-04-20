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
