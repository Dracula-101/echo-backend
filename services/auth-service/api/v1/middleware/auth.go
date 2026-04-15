package middleware

import (
	"auth-service/internal/service/session"
	"errors"
	"net/http"
	"shared/pkg/logger"
	"shared/server/headers"
	"shared/server/middleware"
	"shared/server/response"
)

func SessionAuth(service session.SessionService, sessionCache session.SessionCache, log logger.Logger, skipPaths ...string) middleware.Handler {
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
			log.Debug("Printing the headers",
				logger.String("path", path),
				logger.Any("headers", r.Header),
			)
			log.Debug("SessionAuth middleware: received request",
				logger.String("path", path),
				logger.String("session_id", sessionID),
			)
			if sessionID == "" {
				response.UnauthorizedError(ctx, r, w, "Missing session ID", errors.New("missing session ID in headers"))
				return
			}

			session, err := sessionCache.GetSession(sessionID)
			if err != nil {
				response.UnauthorizedError(ctx, r, w, "Invalid session", errors.New("session not found in cache"))
				log.Debug("SessionAuth middleware: session not found in cache",
					logger.String("path", path),
					logger.String("session_id", sessionID),
				)
				return
			}
			// Check expiration
			if session != nil && session.IsExpired() {
				log.Debug("SessionAuth middleware: session expired",
					logger.String("path", path),
					logger.String("session_id", sessionID),
				)
				response.UnauthorizedError(ctx, r, w, "Session expired", errors.New("session expired"))
				return
			}

			// validate session with auth service
			sessionData, _ := service.GetSessionByID(ctx, sessionID)
			if sessionData == nil {
				response.UnauthorizedError(ctx, r, w, "Invalid session - session not found", errors.New("session not found"))
				log.Debug("SessionAuth middleware: session not found",
					logger.String("path", path),
					logger.String("session_id", sessionID),
				)
				return
			}
			if sessionData.IsExpired() {
				response.UnauthorizedError(ctx, r, w, "Session expired - session not found", errors.New("session expired"))
				log.Debug("SessionAuth middleware: session expired",
					logger.String("path", path),
					logger.String("session_id", sessionID),
				)
				return
			}

			sessionCache.SetSession(ctx, sessionData.UserID, sessionData.ID, sessionData.SessionToken, sessionData.DeviceID, sessionData.ExpiresAt)
			next.ServeHTTP(w, r)
		})
	}
}
