package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Claims represents the authenticated user's claims
type Claims struct {
	UserID   string
	Email    string
	Username string
	Roles    []string
	Metadata map[string]interface{}
}

// Validator defines the interface for token validation
type Validator interface {
	// Validate validates a token and returns the claims
	Validate(ctx context.Context, token string) (*Claims, error)
}

type ctxKey string

const claimsKey ctxKey = "claims"

// WithClaims adds claims to the context
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext extracts claims from the context
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

type AuthInterceptor struct {
	validator Validator
}

func NewAuthInterceptor(validator Validator) *AuthInterceptor {
	return &AuthInterceptor{
		validator: validator,
	}
}

func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		token, err := extractToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		claims, err := i.validator.Validate(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = WithClaims(ctx, claims)
		return handler(ctx, req)
	}
}

func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		token, err := extractToken(ss.Context())
		if err != nil {
			return status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		claims, err := i.validator.Validate(ss.Context(), token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx := WithClaims(ss.Context(), claims)
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

func extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := values[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	return strings.TrimPrefix(token, "Bearer "), nil
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
