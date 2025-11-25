package context

import (
	"context"
)

// Key is a context key type
type Key string

const (
	// KeyConnectionID is the connection ID context key
	KeyConnectionID Key = "connection_id"

	// KeyUserID is the user ID context key
	KeyUserID Key = "user_id"

	// KeySessionID is the session ID context key
	KeySessionID Key = "session_id"

	// KeyRequestID is the request ID context key
	KeyRequestID Key = "request_id"

	// KeyMetadata is the metadata context key
	KeyMetadata Key = "metadata"
)

// WithConnectionID adds connection ID to context
func WithConnectionID(ctx context.Context, connID string) context.Context {
	return context.WithValue(ctx, KeyConnectionID, connID)
}

// GetConnectionID retrieves connection ID from context
func GetConnectionID(ctx context.Context) (string, bool) {
	connID, ok := ctx.Value(KeyConnectionID).(string)
	return connID, ok
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, KeyUserID, userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(KeyUserID).(string)
	return userID, ok
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, KeySessionID, sessionID)
}

// GetSessionID retrieves session ID from context
func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(KeySessionID).(string)
	return sessionID, ok
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, KeyRequestID, requestID)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(KeyRequestID).(string)
	return requestID, ok
}

// WithMetadata adds metadata to context
func WithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	return context.WithValue(ctx, KeyMetadata, metadata)
}

// GetMetadata retrieves metadata from context
func GetMetadata(ctx context.Context) (map[string]interface{}, bool) {
	metadata, ok := ctx.Value(KeyMetadata).(map[string]interface{})
	return metadata, ok
}
