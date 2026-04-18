package cache

import (
	"context"
	"encoding/json"
	"time"
	"user-service/internal/domain"

	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type UserCache interface {
	GetProfile(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError)
	SetProfile(ctx context.Context, userID string, p *domain.Profile) pkgErrors.AppError
	DeleteProfile(ctx context.Context, userID string) pkgErrors.AppError

	GetUserIDByUsername(ctx context.Context, username string) (string, pkgErrors.AppError)
	SetUserIDByUsername(ctx context.Context, username, userID string) pkgErrors.AppError
	DeleteUserIDByUsername(ctx context.Context, username string) pkgErrors.AppError

	GetUserIDByPhone(ctx context.Context, e164 string) (string, pkgErrors.AppError)
	SetUserIDByPhone(ctx context.Context, e164, userID string) pkgErrors.AppError
	DeleteUserIDByPhone(ctx context.Context, e164 string) pkgErrors.AppError

	GetSettings(ctx context.Context, userID string) (*Settings, pkgErrors.AppError)
	SetSettings(ctx context.Context, userID string, s *Settings) pkgErrors.AppError
	DeleteSettings(ctx context.Context, userID string) pkgErrors.AppError

	GetPresence(ctx context.Context, userID string) (*Presence, pkgErrors.AppError)
	SetPresence(ctx context.Context, userID string, p *Presence) pkgErrors.AppError
	DeletePresence(ctx context.Context, userID string) pkgErrors.AppError

	GetContactPresence(ctx context.Context, userID string) ([]*ContactPresenceEntry, pkgErrors.AppError)
	SetContactPresence(ctx context.Context, userID string, entries []*ContactPresenceEntry) pkgErrors.AppError
	DeleteContactPresence(ctx context.Context, userID string) pkgErrors.AppError

	GetBlockedIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError)
	SetBlockedIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError
	DeleteBlockedIDs(ctx context.Context, userID string) pkgErrors.AppError

	GetContactIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError)
	SetContactIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError
	DeleteContactIDs(ctx context.Context, userID string) pkgErrors.AppError

	GetSearchCache(ctx context.Context, key string) (*SearchResult, pkgErrors.AppError)
	SetSearchCache(ctx context.Context, key string, result *SearchResult) pkgErrors.AppError

	GetStatusFeed(ctx context.Context, userID string) ([]*StatusEntry, pkgErrors.AppError)
	SetStatusFeed(ctx context.Context, userID string, entries []*StatusEntry) pkgErrors.AppError
	DeleteStatusFeed(ctx context.Context, userID string) pkgErrors.AppError
}

type userCache struct {
	cache cache.Cache
	log   logger.Logger
}

func NewUserCache(c cache.Cache, log logger.Logger) UserCache {
	return &userCache{cache: c, log: log}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshal[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func unmarshalSlice[T any](data []byte) ([]*T, error) {
	var v []*T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func unmarshalStringSlice(data []byte) ([]string, error) {
	var v []string
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (u *userCache) getJSON(ctx context.Context, key, label string) ([]byte, pkgErrors.AppError) {
	data, err := u.cache.Get(ctx, key)
	if err != nil {
		u.log.Error("cache get failed", logger.String("key", key), logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	return data, nil
}

func (u *userCache) setJSON(ctx context.Context, key string, v any, ttl time.Duration, label string) pkgErrors.AppError {
	data, err := marshal(v)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to marshal "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	if appErr := u.cache.Set(ctx, key, data, ttl); appErr != nil {
		u.log.Error("cache set failed", logger.String("key", key), logger.Error(appErr))
		return pkgErrors.FromError(appErr, pkgErrors.CodeCacheError, "failed to set "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	return nil
}

func (u *userCache) del(ctx context.Context, key, label string) pkgErrors.AppError {
	if err := u.cache.Delete(ctx, key); err != nil {
		u.log.Error("cache delete failed", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to delete "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	return nil
}

func (u *userCache) getString(ctx context.Context, key, label string) (string, pkgErrors.AppError) {
	val, err := u.cache.GetString(ctx, key)
	if err != nil {
		u.log.Error("cache get string failed", logger.String("key", key), logger.Error(err))
		return "", pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	return val, nil
}

func (u *userCache) setString(ctx context.Context, key, val string, ttl time.Duration, label string) pkgErrors.AppError {
	if err := u.cache.SetString(ctx, key, val, ttl); err != nil {
		u.log.Error("cache set string failed", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to set "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	return nil
}

// ── profile ───────────────────────────────────────────────────────────────────

func (u *userCache) GetProfile(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, ProfilePrefix+userID, "profile")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	p, err := unmarshal[domain.Profile](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal profile").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return p, nil
}

func (u *userCache) SetProfile(ctx context.Context, userID string, p *domain.Profile) pkgErrors.AppError {
	return u.setJSON(ctx, ProfilePrefix+userID, p, TTLProfile, "profile")
}

func (u *userCache) DeleteProfile(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, ProfilePrefix+userID, "profile")
}

// ── username → user_id ────────────────────────────────────────────────────────

func (u *userCache) GetUserIDByUsername(ctx context.Context, username string) (string, pkgErrors.AppError) {
	return u.getString(ctx, ProfileUsernamePrefix+username, "username mapping")
}

func (u *userCache) SetUserIDByUsername(ctx context.Context, username, userID string) pkgErrors.AppError {
	return u.setString(ctx, ProfileUsernamePrefix+username, userID, TTLProfileUsername, "username mapping")
}

func (u *userCache) DeleteUserIDByUsername(ctx context.Context, username string) pkgErrors.AppError {
	return u.del(ctx, ProfileUsernamePrefix+username, "username mapping")
}

// ── phone → user_id ───────────────────────────────────────────────────────────

func (u *userCache) GetUserIDByPhone(ctx context.Context, e164 string) (string, pkgErrors.AppError) {
	return u.getString(ctx, ProfilePhonePrefix+e164, "phone mapping")
}

func (u *userCache) SetUserIDByPhone(ctx context.Context, e164, userID string) pkgErrors.AppError {
	return u.setString(ctx, ProfilePhonePrefix+e164, userID, TTLProfilePhone, "phone mapping")
}

func (u *userCache) DeleteUserIDByPhone(ctx context.Context, e164 string) pkgErrors.AppError {
	return u.del(ctx, ProfilePhonePrefix+e164, "phone mapping")
}

// ── settings ──────────────────────────────────────────────────────────────────

func (u *userCache) GetSettings(ctx context.Context, userID string) (*Settings, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, SettingsPrefix+userID, "settings")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	s, err := unmarshal[Settings](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal settings").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return s, nil
}

func (u *userCache) SetSettings(ctx context.Context, userID string, s *Settings) pkgErrors.AppError {
	return u.setJSON(ctx, SettingsPrefix+userID, s, TTLSettings, "settings")
}

func (u *userCache) DeleteSettings(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, SettingsPrefix+userID, "settings")
}

// ── presence ──────────────────────────────────────────────────────────────────

func (u *userCache) GetPresence(ctx context.Context, userID string) (*Presence, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, PresencePrefix+userID, "presence")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	p, err := unmarshal[Presence](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal presence").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return p, nil
}

func (u *userCache) SetPresence(ctx context.Context, userID string, p *Presence) pkgErrors.AppError {
	return u.setJSON(ctx, PresencePrefix+userID, p, TTLPresence, "presence")
}

func (u *userCache) DeletePresence(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, PresencePrefix+userID, "presence")
}

// ── contact presence ──────────────────────────────────────────────────────────

func (u *userCache) GetContactPresence(ctx context.Context, userID string) ([]*ContactPresenceEntry, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, PresenceContactPrefix+userID, "contact presence")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	entries, err := unmarshalSlice[ContactPresenceEntry](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal contact presence").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return entries, nil
}

func (u *userCache) SetContactPresence(ctx context.Context, userID string, entries []*ContactPresenceEntry) pkgErrors.AppError {
	return u.setJSON(ctx, PresenceContactPrefix+userID, entries, TTLPresenceContact, "contact presence")
}

func (u *userCache) DeleteContactPresence(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, PresenceContactPrefix+userID, "contact presence")
}

// ── blocked IDs ───────────────────────────────────────────────────────────────

func (u *userCache) GetBlockedIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, BlockedPrefix+userID, "blocked IDs")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	ids, err := unmarshalStringSlice(data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal blocked IDs").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return ids, nil
}

func (u *userCache) SetBlockedIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError {
	return u.setJSON(ctx, BlockedPrefix+userID, ids, TTLBlocked, "blocked IDs")
}

func (u *userCache) DeleteBlockedIDs(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, BlockedPrefix+userID, "blocked IDs")
}

// ── contact IDs ───────────────────────────────────────────────────────────────

func (u *userCache) GetContactIDs(ctx context.Context, userID string) ([]string, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, ContactIDsPrefix+userID, "contact IDs")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	ids, err := unmarshalStringSlice(data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal contact IDs").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return ids, nil
}

func (u *userCache) SetContactIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError {
	return u.setJSON(ctx, ContactIDsPrefix+userID, ids, TTLContactIDs, "contact IDs")
}

func (u *userCache) DeleteContactIDs(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, ContactIDsPrefix+userID, "contact IDs")
}

// ── search cache ──────────────────────────────────────────────────────────────

func (u *userCache) GetSearchCache(ctx context.Context, key string) (*SearchResult, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, SearchCachePrefix+key, "search cache")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	r, err := unmarshal[SearchResult](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal search cache").
			WithService("user-service").WithDetail("key", key)
	}
	return r, nil
}

func (u *userCache) SetSearchCache(ctx context.Context, key string, result *SearchResult) pkgErrors.AppError {
	return u.setJSON(ctx, SearchCachePrefix+key, result, TTLSearchCache, "search cache")
}

// ── status feed ───────────────────────────────────────────────────────────────

func (u *userCache) GetStatusFeed(ctx context.Context, userID string) ([]*StatusEntry, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, StatusFeedPrefix+userID, "status feed")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	entries, err := unmarshalSlice[StatusEntry](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal status feed").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return entries, nil
}

func (u *userCache) SetStatusFeed(ctx context.Context, userID string, entries []*StatusEntry) pkgErrors.AppError {
	return u.setJSON(ctx, StatusFeedPrefix+userID, entries, TTLStatusFeed, "status feed")
}

func (u *userCache) DeleteStatusFeed(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, StatusFeedPrefix+userID, "status feed")
}
