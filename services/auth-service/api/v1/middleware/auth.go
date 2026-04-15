package middleware

import (
	"auth-service/internal/service/session"
	"net/http"
	"shared/server/headers"
	"shared/server/middleware"
	"shared/server/response"
)

func SessionAuth(service session.SessionService, sessionCache session.SessionCache, skipPaths ...string) middleware.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			sessionID := r.Header.Get(headers.XSessionID)
			if sessionID == "" {
				http.Error(w, "Missing session ID", http.StatusUnauthorized)
				return
			}
			session, err := sessionCache.GetSession(sessionID)
			if err != nil {
				http.Error(w, "Invalid session", http.StatusUnauthorized)
				return
			}
			// Check expiration
			if session != nil && session.IsExpired() {
				response.UnauthorizedError(ctx, r, w, "Session expired", nil)
				return
			}

			// validate session with auth service
			sessionData, _ := service.GetSessionByID(ctx, sessionID)
			if sessionData == nil {
				response.UnauthorizedError(ctx, r, w, "Invalid session", nil)
				return
			}
			if sessionData.IsExpired() {
				response.UnauthorizedError(ctx, r, w, "Session expired", nil)
				return
			}

			sessionCache.SetSession(ctx, sessionData.UserID, sessionData.ID, sessionData.SessionToken, sessionData.DeviceID, sessionData.ExpiresAt)
			next.ServeHTTP(w, r)
		})
	}
}
