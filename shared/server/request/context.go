package request

import (
	"context"
	sContext "shared/server/context"

	"github.com/google/uuid"
)

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, sContext.UserIDKey, userID)
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(sContext.UserIDKey).(string)
	return userID, ok
}

func GetUserIDUUIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userIDStr, ok := ctx.Value(sContext.UserIDKey).(string)
	if !ok {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sContext.SessionIDKey, sessionID)
}

// GetSessionIDFromContext retrieves session ID from context
func GetSessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sContext.SessionIDKey).(string)
	return sessionID, ok
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, sContext.RequestIDKey, requestID)
}

// GetRequestIDFromContext retrieves request ID from context
func GetRequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(sContext.RequestIDKey).(string)
	return requestID, ok
}

// WithClientIP adds client IP to context
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, sContext.ClientIPKey, ip)
}

// GetClientIPFromContext retrieves client IP from context
func GetClientIPFromContext(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(sContext.ClientIPKey).(string)
	return ip, ok
}
