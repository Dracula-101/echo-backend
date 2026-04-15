package middleware

import (
	"auth-service/internal/service/session"
	"errors"
	"net/http"
	"shared/server/headers"
	"shared/server/middleware"
	"shared/server/response"
)

func SessionAuth(service session.SessionService, sessionCache session.SessionCache, skipPaths ...string) middleware.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			path := r.URL.Path
			for _, skipPath := range skipPaths {
				if path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			ctx := r.Context()
			sessionID := r.Header.Get(headers.XSessionID)
			if sessionID == "" {
				response.UnauthorizedError(ctx, r, w, "Missing session ID", errors.New("missing session ID in headers"))
				return
			}
			session, err := sessionCache.GetSession(sessionID)
			if err != nil {
				response.UnauthorizedError(ctx, r, w, "Invalid session", errors.New("session not found in cache"))
				return
			}
			// Check expiration
			if session != nil && session.IsExpired() {
				response.UnauthorizedError(ctx, r, w, "Session expired", errors.New("session expired"))
				return
			}

			// validate session with auth service
			sessionData, _ := service.GetSessionByID(ctx, sessionID)
			if sessionData == nil {
				response.UnauthorizedError(ctx, r, w, "Invalid session - session not found", errors.New("session not found"))
				return
			}
			if sessionData.IsExpired() {
				response.UnauthorizedError(ctx, r, w, "Session expired - session not found", errors.New("session expired"))
				return
			}

			sessionCache.SetSession(ctx, sessionData.UserID, sessionData.ID, sessionData.SessionToken, sessionData.DeviceID, sessionData.ExpiresAt)
			next.ServeHTTP(w, r)
		})
	}
}
