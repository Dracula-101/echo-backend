package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

	GetUserIDByUsername(ctx context.Context, username string) (*string, pkgErrors.AppError)
	SetUserIDByUsername(ctx context.Context, username, userID string) pkgErrors.AppError
	DeleteUserIDByUsername(ctx context.Context, username string) pkgErrors.AppError

	GetUserIDByPhone(ctx context.Context, e164 string) (*string, pkgErrors.AppError)
	SetUserIDByPhone(ctx context.Context, e164, userID string) pkgErrors.AppError
	DeleteUserIDByPhone(ctx context.Context, e164 string) pkgErrors.AppError

	GetSettings(ctx context.Context, userID string) (*Settings, pkgErrors.AppError)
	SetSettings(ctx context.Context, userID string, s *Settings) pkgErrors.AppError
	DeleteSettings(ctx context.Context, userID string) pkgErrors.AppError

	GetFullSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError)
	SetFullSettings(ctx context.Context, userID string, s *domain.Settings) pkgErrors.AppError
	DeleteFullSettings(ctx context.Context, userID string) pkgErrors.AppError

	GetPresence(ctx context.Context, userID string) (*Presence, pkgErrors.AppError)
	GetPresences(ctx context.Context, userIDs []string) (map[string]*Presence, pkgErrors.AppError)
	SetPresence(ctx context.Context, userID string, p *Presence) pkgErrors.AppError
	DeletePresence(ctx context.Context, userID string) pkgErrors.AppError

	GetContactPresence(ctx context.Context, userID string) ([]*ContactPresenceEntry, pkgErrors.AppError)
	SetContactPresence(ctx context.Context, userID string, entries []*ContactPresenceEntry) pkgErrors.AppError
	DeleteContactPresence(ctx context.Context, userID string) pkgErrors.AppError

	GetBlockedIDs(ctx context.Context, userID string) (*[]string, pkgErrors.AppError)
	AddBlockedID(ctx context.Context, userID string, blockedID string) pkgErrors.AppError
	RemoveBlockedID(ctx context.Context, userID string, blockedID string) pkgErrors.AppError
	SetBlockedIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError
	DeleteBlockedIDs(ctx context.Context, userID string) pkgErrors.AppError

	GetContactIDs(ctx context.Context, userID string) (*[]string, pkgErrors.AppError)
	AddContactID(ctx context.Context, userID string, contactID string) pkgErrors.AppError
	RemoveContact(ctx context.Context, userID string, contactID string) pkgErrors.AppError
	SetContactIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError
	DeleteContactIDs(ctx context.Context, userID string) pkgErrors.AppError

	GetSearchCache(ctx context.Context, key string) (*SearchResult, pkgErrors.AppError)
	SetSearchCache(ctx context.Context, key string, result *SearchResult) pkgErrors.AppError

	GetStatusFeed(ctx context.Context, userID string) (*[]*StatusEntry, pkgErrors.AppError)
	SetStatusFeed(ctx context.Context, userID string, entries []*StatusEntry) pkgErrors.AppError
	DeleteStatusFeed(ctx context.Context, userID string) pkgErrors.AppError

	GetPrivacy(ctx context.Context, viewerID, subjectID string) (*PrivacyResult, pkgErrors.AppError)
	SetPrivacy(ctx context.Context, viewerID, subjectID string, r *PrivacyResult) pkgErrors.AppError
	DeletePrivacy(ctx context.Context, viewerID, subjectID string) pkgErrors.AppError
	GetPrivacyVersion(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	IncrPrivacyVersion(ctx context.Context, userID string) (int64, pkgErrors.AppError)

	AcquireContactLock(ctx context.Context, userA, userB string) (bool, pkgErrors.AppError)
	ReleaseContactLock(ctx context.Context, userA, userB string) pkgErrors.AppError
	AcquireBlockLock(ctx context.Context, userA, userB string) (bool, pkgErrors.AppError)
	ReleaseBlockLock(ctx context.Context, userA, userB string) pkgErrors.AppError
}

type userCache struct {
	cache cache.Cache
	log   logger.Logger
}

func NewUserCache(c cache.Cache, log logger.Logger) UserCache {
	return &userCache{cache: c, log: log}
}

const (
	PrivacyPrefix        = "privacy:"
	PrivacyVersionPrefix = "privacy_version:"
	LockContactPrefix    = "lock:contact:"
	LockBlockPrefix      = "lock:block:"

	TTLPrivacy = 120 * time.Second
	TTLLock    = 10 * time.Second
)

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

func canonicalPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func (u *userCache) getJSON(ctx context.Context, key, label string) ([]byte, pkgErrors.AppError) {
	data, err := u.cache.Get(ctx, key)
	if err != nil {
		if err.Code() == pkgErrors.CodeNotFound {
			return nil, nil
		}
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

func (u *userCache) getString(ctx context.Context, key, label string) (*string, pkgErrors.AppError) {
	val, err := u.cache.GetString(ctx, key)
	if err != nil {
		if err.Code() == pkgErrors.CodeNotFound {
			return nil, nil
		}
		u.log.Error("cache get string failed", logger.String("key", key), logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get "+label).
			WithService("user-service").
			WithDetail("key", key)
	}
	return &val, nil
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

func (u *userCache) GetUserIDByUsername(ctx context.Context, username string) (*string, pkgErrors.AppError) {
	return u.getString(ctx, ProfileUsernamePrefix+username, "username mapping")
}

func (u *userCache) SetUserIDByUsername(ctx context.Context, username, userID string) pkgErrors.AppError {
	return u.setString(ctx, ProfileUsernamePrefix+username, userID, TTLProfileUsername, "username mapping")
}

func (u *userCache) DeleteUserIDByUsername(ctx context.Context, username string) pkgErrors.AppError {
	return u.del(ctx, ProfileUsernamePrefix+username, "username mapping")
}

// ── phone → user_id ───────────────────────────────────────────────────────────

func (u *userCache) GetUserIDByPhone(ctx context.Context, e164 string) (*string, pkgErrors.AppError) {
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

// ── full settings ─────────────────────────────────────────────────────────────

func (u *userCache) GetFullSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, SettingsFullPrefix+userID, "full settings")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	s, err := unmarshal[domain.Settings](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal full settings").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return s, nil
}

func (u *userCache) SetFullSettings(ctx context.Context, userID string, s *domain.Settings) pkgErrors.AppError {
	return u.setJSON(ctx, SettingsFullPrefix+userID, s, TTLSettings, "full settings")
}

func (u *userCache) DeleteFullSettings(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, SettingsFullPrefix+userID, "full settings")
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

func (u *userCache) GetPresences(ctx context.Context, userIDs []string) (map[string]*Presence, pkgErrors.AppError) {
	if len(userIDs) == 0 {
		return map[string]*Presence{}, nil
	}

	keys := make([]string, 0, len(userIDs))
	keyToUserID := make(map[string]string, len(userIDs))
	for _, userID := range userIDs {
		key := PresencePrefix + userID
		keys = append(keys, key)
		keyToUserID[key] = userID
	}

	items, err := u.cache.GetMulti(ctx, keys)
	if err != nil {
		u.log.Error("cache get multi presence failed", logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get presence in batch").
			WithService("user-service")
	}

	result := make(map[string]*Presence, len(items))
	for key, raw := range items {
		presence, unmarshalErr := unmarshal[Presence](raw)
		if unmarshalErr != nil {
			u.log.Warn("failed to unmarshal cached presence entry",
				logger.String("key", key),
				logger.Error(unmarshalErr),
			)
			continue
		}
		userID := keyToUserID[key]
		result[userID] = presence
	}

	return result, nil
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

func (u *userCache) GetBlockedIDs(ctx context.Context, userID string) (*[]string, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, BlockedPrefix+userID, "blocked IDs")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	ids, err := unmarshalStringSlice(data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal blocked IDs").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return &ids, nil
}

func (u *userCache) RemoveBlockedID(ctx context.Context, userID string, blockedID string) pkgErrors.AppError {
	idsPtr, err := u.GetBlockedIDs(ctx, userID)
	if err != nil {
		return err
	}
	if idsPtr == nil {
		return nil
	}
	var ids []string
	for _, id := range *idsPtr {
		if id != blockedID {
			ids = append(ids, id)
		}
	}
	return u.SetBlockedIDs(ctx, userID, ids)
}

func (u *userCache) AddBlockedID(ctx context.Context, userID string, blockedID string) pkgErrors.AppError {
	idsPtr, err := u.GetBlockedIDs(ctx, userID)
	if err != nil {
		return err
	}
	var ids []string
	if idsPtr != nil {
		ids = *idsPtr
	}
	// Check for duplicates before adding
	for _, id := range ids {
		if id == blockedID {
			return nil // Already exists, no need to add
		}
	}
	ids = append(ids, blockedID)
	return u.SetBlockedIDs(ctx, userID, ids)
}

func (u *userCache) SetBlockedIDs(ctx context.Context, userID string, ids []string) pkgErrors.AppError {
	return u.setJSON(ctx, BlockedPrefix+userID, ids, TTLBlocked, "blocked IDs")
}

func (u *userCache) DeleteBlockedIDs(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, BlockedPrefix+userID, "blocked IDs")
}

// ── contact IDs ───────────────────────────────────────────────────────────────

func (u *userCache) GetContactIDs(ctx context.Context, userID string) (*[]string, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, ContactIDsPrefix+userID, "contact IDs")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	ids, err := unmarshalStringSlice(data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal contact IDs").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return &ids, nil
}

func (u *userCache) AddContactID(ctx context.Context, userID string, contactID string) pkgErrors.AppError {
	idsPtr, err := u.GetContactIDs(ctx, userID)
	if err != nil {
		return err
	}
	var ids []string
	if idsPtr != nil {
		ids = *idsPtr
	}
	// Check for duplicates before adding
	for _, id := range ids {
		if id == contactID {
			return nil // Already exists, no need to add
		}
	}
	ids = append(ids, contactID)
	return u.SetContactIDs(ctx, userID, ids)
}

func (u *userCache) RemoveContactID(ctx context.Context, userID string, contactID string) pkgErrors.AppError {
	idsPtr, err := u.GetContactIDs(ctx, userID)
	if err != nil {
		return err
	}
	if idsPtr == nil {
		return nil // Nothing to remove
	}
	var ids []string
	for _, id := range *idsPtr {
		if id != contactID {
			ids = append(ids, id)
		}
	}
	return u.SetContactIDs(ctx, userID, ids)
}

func (u *userCache) RemoveContact(ctx context.Context, userID string, contactID string) pkgErrors.AppError {
	if err := u.RemoveContactID(ctx, userID, contactID); err != nil {
		return err
	}

	if _, err := u.IncrPrivacyVersion(ctx, userID); err != nil {
		return err
	}

	return nil
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

func (u *userCache) GetStatusFeed(ctx context.Context, userID string) (*[]*StatusEntry, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, StatusFeedPrefix+userID, "status feed")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	entries, err := unmarshalSlice[StatusEntry](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal status feed").
			WithService("user-service").WithDetail("user_id", userID)
	}
	return &entries, nil
}

func (u *userCache) SetStatusFeed(ctx context.Context, userID string, entries []*StatusEntry) pkgErrors.AppError {
	return u.setJSON(ctx, StatusFeedPrefix+userID, entries, TTLStatusFeed, "status feed")
}

func (u *userCache) DeleteStatusFeed(ctx context.Context, userID string) pkgErrors.AppError {
	return u.del(ctx, StatusFeedPrefix+userID, "status feed")
}

func privacyKey(viewerID, subjectID string) string {
	return fmt.Sprintf("%s%s:%s", PrivacyPrefix, viewerID, subjectID)
}

func (u *userCache) GetPrivacy(ctx context.Context, viewerID, subjectID string) (*PrivacyResult, pkgErrors.AppError) {
	data, appErr := u.getJSON(ctx, privacyKey(viewerID, subjectID), "privacy result")
	if appErr != nil || data == nil {
		return nil, appErr
	}
	r, err := unmarshal[PrivacyResult](data)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to unmarshal privacy result").
			WithService("user-service").
			WithDetail("viewer_id", viewerID).
			WithDetail("subject_id", subjectID)
	}
	return r, nil
}

func (u *userCache) SetPrivacy(ctx context.Context, viewerID, subjectID string, r *PrivacyResult) pkgErrors.AppError {
	return u.setJSON(ctx, privacyKey(viewerID, subjectID), r, TTLPrivacy, "privacy result")
}

func (u *userCache) DeletePrivacy(ctx context.Context, viewerID, subjectID string) pkgErrors.AppError {
	return u.del(ctx, privacyKey(viewerID, subjectID), "privacy result")
}

func (u *userCache) GetPrivacyVersion(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	val, err := u.cache.GetString(ctx, PrivacyVersionPrefix+userID)
	if err != nil {
		if err.Code() == pkgErrors.CodeNotFound {
			return 0, nil
		}
		return 0, err
	}
	v, parseErr := strconv.ParseInt(val, 10, 64)
	if parseErr != nil {
		return 0, pkgErrors.FromError(parseErr, pkgErrors.CodeCacheError, "failed to parse privacy version").
			WithService("user-service").
			WithDetail("user_id", userID)
	}
	return v, nil
}

func (u *userCache) IncrPrivacyVersion(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	newVal, err := u.cache.Increment(ctx, PrivacyVersionPrefix+userID, 1)
	if err != nil {
		u.log.Error("failed to increment privacy version",
			logger.String("user_id", userID),
			logger.Error(err))
		return 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to increment privacy version").
			WithService("user-service").
			WithDetail("user_id", userID)
	}
	return newVal, nil
}

func (u *userCache) AcquireContactLock(ctx context.Context, userA, userB string) (bool, pkgErrors.AppError) {
	lo, hi := canonicalPair(userA, userB)
	key := fmt.Sprintf("%s%s:%s", LockContactPrefix, lo, hi)
	acquired, err := u.cache.AcquireLock(ctx, key, TTLLock)
	if err != nil {
		u.log.Error("contact lock acquire failed", logger.String("key", key), logger.Error(err))
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire contact lock").
			WithService("user-service").
			WithDetail("key", key)
	}
	return acquired, nil
}

func (u *userCache) ReleaseContactLock(ctx context.Context, userA, userB string) pkgErrors.AppError {
	lo, hi := canonicalPair(userA, userB)
	key := fmt.Sprintf("%s%s:%s", LockContactPrefix, lo, hi)
	if err := u.cache.ReleaseLock(ctx, key); err != nil {
		u.log.Warn("contact lock release failed", logger.String("key", key), logger.Error(err))
	}
	return nil
}

func (u *userCache) AcquireBlockLock(ctx context.Context, userA, userB string) (bool, pkgErrors.AppError) {
	lo, hi := canonicalPair(userA, userB)
	key := fmt.Sprintf("%s%s:%s", LockBlockPrefix, lo, hi)
	acquired, err := u.cache.AcquireLock(ctx, key, TTLLock)
	if err != nil {
		u.log.Error("block lock acquire failed", logger.String("key", key), logger.Error(err))
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to acquire block lock").
			WithService("user-service").
			WithDetail("key", key)
	}
	return acquired, nil
}

func (u *userCache) ReleaseBlockLock(ctx context.Context, userA, userB string) pkgErrors.AppError {
	lo, hi := canonicalPair(userA, userB)
	key := fmt.Sprintf("%s%s:%s", LockBlockPrefix, lo, hi)
	if err := u.cache.ReleaseLock(ctx, key); err != nil {
		u.log.Warn("block lock release failed", logger.String("key", key), logger.Error(err))
	}
	return nil
}
