// Package idempotency implements the X-Idempotency-Key flow for write
// endpoints (send message, forward message, etc.).
//
// The pattern is a three-step Redis dance keyed on lock:{scope}:{key}:
//
//  1. CheckAndAcquire: SETNX the key with the in-flight sentinel "1".
//     If acquired, the caller owns the request. If the key already exists,
//     either return the stored response payload (previous success) or
//     report 409 conflict (concurrent request still processing).
//
//  2. Store: after the business write commits, overwrite the sentinel with
//     the response payload JSON. Refreshes TTL.
//
//  3. Release: on failure after acquire, delete the sentinel so the client
//     can retry without waiting for the TTL.
//
// Callers must invoke exactly one of Store or Release after a successful
// CheckAndAcquire — typically Release in a deferred error path and Store on
// the happy path. Forgetting both leaves the key blocked for the full TTL.
package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	msgError "echo-backend/services/message-service/internal/error"

	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Scope segregates idempotency keys per operation type. Two different
// endpoints can safely use the same client-supplied key.
const (
	ScopeSendMessage    = "msg"
	ScopeForwardMessage = "fwd"
	ScopeCreateConv     = "convo"
	ScopeAcceptInvite   = "invite"
)

const (
	// DefaultTTL matches the spec for lock:msg:{key}.
	DefaultTTL = 5 * time.Minute
	// inFlightSentinel is what AcquireLock writes into the key. Any value
	// not equal to this is treated as a stored response payload.
	inFlightSentinel = "1"
)

// Result is the outcome of CheckAndAcquire.
//
// Three combinations are valid:
//
//	Hit=false                       caller acquired the key, proceed
//	Hit=true, InFlight=true         concurrent request in progress, return 409
//	Hit=true, Payload!=nil          previous success, replay payload as 200
type Result struct {
	Hit      bool
	InFlight bool
	Payload  json.RawMessage
}

type Manager interface {
	// CheckAndAcquire validates the key, then atomically claims it via SETNX.
	// Returns a Result describing what the caller should do next.
	CheckAndAcquire(ctx context.Context, scope, key string) (*Result, *msgError.MsgError)

	// Store overwrites the in-flight sentinel with the cacheable response
	// payload. Refreshes TTL to DefaultTTL. Call after the business write
	// commits successfully.
	Store(ctx context.Context, scope, key string, payload []byte) *msgError.MsgError

	// Release deletes the in-flight sentinel so the client can retry.
	// Idempotent — safe to call even if Store has already been called or
	// the key has already expired.
	Release(ctx context.Context, scope, key string) *msgError.MsgError
}

type manager struct {
	cache cache.Cache
	log   logger.Logger
	ttl   time.Duration
}

func NewManager(c cache.Cache, log logger.Logger) Manager {
	return &manager{cache: c, log: log, ttl: DefaultTTL}
}

func (m *manager) CheckAndAcquire(ctx context.Context, scope, key string) (*Result, *msgError.MsgError) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	cacheKey := buildKey(scope, key)

	acquired, err := m.cache.AcquireLock(ctx, cacheKey, m.ttl)
	if err != nil {
		m.log.Error("idempotency acquire failed",
			logger.String("scope", scope),
			logger.String("cache_key", cacheKey),
			logger.Error(err),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to check idempotency key",
			Code:    msgError.CodeCacheError,
			Error: pkgErrors.FromError(err, msgError.CodeCacheError, "idempotency acquire failed").
				WithService(msgError.ServiceName).
				WithDetail("cache_key", cacheKey),
		}
	}

	if acquired {
		return &Result{Hit: false}, nil
	}

	// Key already existed. Read the current value to decide between
	// in-flight (return 409) and stored payload (replay 200).
	existing, getErr := m.cache.GetString(ctx, cacheKey)
	if getErr != nil {
		// The lock could have expired between SETNX and GET (unlikely but
		// possible). Treat as in-flight — the client can retry.
		m.log.Warn("idempotency key vanished between acquire and read",
			logger.String("cache_key", cacheKey),
		)
		return &Result{Hit: true, InFlight: true}, nil
	}

	if existing == inFlightSentinel {
		return &Result{Hit: true, InFlight: true}, nil
	}

	return &Result{Hit: true, Payload: json.RawMessage(existing)}, nil
}

func (m *manager) Store(ctx context.Context, scope, key string, payload []byte) *msgError.MsgError {
	if err := validateKey(key); err != nil {
		return err
	}
	if len(payload) == 0 {
		return &msgError.MsgError{
			Message: "Idempotency payload is empty",
			Code:    msgError.CodeInternalError,
			Error:   pkgErrors.New(msgError.CodeInternalError, "idempotency payload is empty").WithService(msgError.ServiceName),
		}
	}

	cacheKey := buildKey(scope, key)
	if setErr := m.cache.SetString(ctx, cacheKey, string(payload), m.ttl); setErr != nil {
		m.log.Error("idempotency store failed",
			logger.String("cache_key", cacheKey),
			logger.Error(setErr),
		)
		return &msgError.MsgError{
			Message: "Failed to store idempotency result",
			Code:    msgError.CodeCacheError,
			Error: pkgErrors.FromError(setErr, msgError.CodeCacheError, "idempotency store failed").
				WithService(msgError.ServiceName).
				WithDetail("cache_key", cacheKey),
		}
	}
	return nil
}

func (m *manager) Release(ctx context.Context, scope, key string) *msgError.MsgError {
	if err := validateKey(key); err != nil {
		return err
	}

	cacheKey := buildKey(scope, key)
	if delErr := m.cache.Delete(ctx, cacheKey); delErr != nil {
		// Best-effort. Log but don't propagate — a leaked sentinel only
		// blocks retries until TTL, which is bounded.
		m.log.Warn("idempotency release failed",
			logger.String("cache_key", cacheKey),
			logger.Error(delErr),
		)
	}
	return nil
}

func validateKey(key string) *msgError.MsgError {
	if key == "" {
		return &msgError.MsgError{
			Message: "X-Idempotency-Key header is required",
			Code:    msgError.CodeIdempotencyKeyRequired,
			Error:   pkgErrors.New(string(msgError.CodeIdempotencyKeyRequired), "missing X-Idempotency-Key header").WithService(msgError.ServiceName),
		}
	}
	if _, err := uuid.Parse(key); err != nil {
		return &msgError.MsgError{
			Message: "X-Idempotency-Key must be a UUID",
			Code:    msgError.CodeIdempotencyKeyInvalid,
			Error: pkgErrors.FromError(errors.New("invalid uuid"), string(msgError.CodeIdempotencyKeyInvalid), "idempotency key is not a valid UUID").
				WithService(msgError.ServiceName),
		}
	}
	return nil
}

func buildKey(scope, key string) string {
	return "lock:" + scope + ":" + key
}
