package app

import (
	"context"
	"fmt"
	"time"

	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/logger"
	prommetrics "shared/pkg/monitoring/metrics/prometheus"
	coreMiddleware "shared/server/middleware"
	"shared/server/request"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"

	conversationHandler "echo-backend/services/message-service/api/v1/handler/conversation"
	messageHandler "echo-backend/services/message-service/api/v1/handler/message"
	"echo-backend/services/message-service/api/v1/middleware"
	ratelimiter "echo-backend/services/message-service/api/v1/rate_limiter"
	"echo-backend/services/message-service/internal/config"
	"echo-backend/services/message-service/internal/health"
	healthCheckers "echo-backend/services/message-service/internal/health/checkers"
)

func buildHTTPServer(
	cfg *config.Config,
	mh *messageHandler.MessageHandler,
	ch *conversationHandler.ConversationHandler,
	rl ratelimiter.RateLimiter,
	db database.Database,
	cacheClient cache.Cache,
	log logger.Logger,
) (*server.Server, error) {
	healthMgr := buildHealthManager(db, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	r := buildRouter(mh, ch, healthHandler, rl, log)

	return server.New(&server.Config{
		Port:            cfg.Server.Port,
		Host:            cfg.Server.Host,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		MaxHeaderBytes:  cfg.Server.MaxHeaderBytes,
		Handler:         r.Mux(),
	}, log)
}

func buildHealthManager(db database.Database, cacheClient cache.Cache, cfg *config.Config) *health.Manager {
	mgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	if db != nil {
		mgr.RegisterChecker(healthCheckers.NewDatabaseChecker(db))
	}
	if cfg.Cache.Enabled && cacheClient != nil {
		mgr.RegisterChecker(healthCheckers.NewCacheChecker(cacheClient))
	}
	return mgr
}

func buildRouter(
	mh *messageHandler.MessageHandler,
	ch *conversationHandler.ConversationHandler,
	healthHandler *health.Handler,
	rl ratelimiter.RateLimiter,
	log logger.Logger,
) *router.Router {
	builder := router.NewBuilder().
		WithHealthEndpoint("/health", func(rh request.RequestHandler) {
			healthHandler.Health(rh.Writer(), rh.Request())
		}).
		WithMetricsEndpoint("/metrics", func(rh request.RequestHandler) {
			prommetrics.Handler().ServeHTTP(rh.Writer(), rh.Request())
		}).
		WithNotFoundHandler(func(rh request.RequestHandler) {
			response.NotFoundError(rh.Context(), rh.Request(), rh.Writer(), fmt.Sprintf("Endpoint %s not found", rh.Request().URL.Path))
		}).
		WithMethodNotAllowedHandler(func(rh request.RequestHandler) {
			response.MethodNotAllowedError(rh.Context(), rh.Request(), rh.Writer())
		}).
		WithEarlyMiddleware(
			router.Middleware(coreMiddleware.Timeout(30*time.Second)),
			router.Middleware(coreMiddleware.BodyLimit(10*1024*1024)),
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(coreMiddleware.RateLimit(coreMiddleware.RateLimitConfig{
				RequestsPerWindow: 100,
				Window:            time.Minute,
			})),
			router.Middleware(coreMiddleware.InterceptUserId()),
			router.Middleware(coreMiddleware.InterceptSessionId()),
			router.Middleware(coreMiddleware.InterceptSessionToken()),
			router.Middleware(coreMiddleware.InterceptIdempotencyKey()),
		).
		WithLateMiddleware(
			router.Middleware(coreMiddleware.Recovery(log)),
			router.Middleware(coreMiddleware.RequestCompletedLogger(log)),
		)

	return registerAPIRoutes(builder, mh, ch, rl).Build()
}

type messageRateLimits struct {
	send   coreMiddleware.Handler
	react  coreMiddleware.Handler
	report coreMiddleware.Handler
	search coreMiddleware.Handler
}

func registerAPIRoutes(
	builder *router.Builder,
	mh *messageHandler.MessageHandler,
	ch *conversationHandler.ConversationHandler,
	rl ratelimiter.RateLimiter,
) *router.Builder {
	userKey := func(req request.RequestHandler) string {
		userID, _ := request.GetUserIDUUIDFromContext(req.Context())
		return userID.String()
	}
	userConvKey := func(req request.RequestHandler) string {
		userID, _ := request.GetUserIDUUIDFromContext(req.Context())
		return userID.String() + "|" + req.PathParam("conversation_id")
	}

	createConvRateLimit := middleware.RateLimit(
		func(ctx context.Context, key string, limit int64) (bool, int64, error) {
			return rl.CheckCreateConvRateLimit(ctx, key)
		},
		userKey,
		ratelimiter.RLCreateConvLimit,
	)
	limits := messageRateLimits{
		send: middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.CheckSendRateLimit(ctx, key)
			}, userKey, ratelimiter.RLSendLimit,
		),
		react: middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.CheckReactRateLimit(ctx, key)
			}, userKey, ratelimiter.RLReactLimit,
		),
		report: middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.CheckReportRateLimit(ctx, key)
			}, userKey, ratelimiter.RLReportLimit,
		),
		search: middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				userID, convID, ok := splitUserConvKey(key)
				if !ok {
					return true, 0, nil
				}
				return rl.CheckSearchRateLimit(ctx, userID, convID)
			}, userConvKey, ratelimiter.RLSearchLimit,
		),
	}

	return builder.WithRoutes(func(r *router.Router) {
		registerConversationRoutes(r, ch, createConvRateLimit)
		registerMessageRoutes(r, mh, limits)
		registerPollRoutes(r, mh)
		registerInviteRoutes(r, ch)
	})
}

// registerConversationRoutes wires endpoints under /conversations.
//
// Note on ordering: literal segments (e.g. "/conversations/me") must be
// registered before parameterized ones (e.g. "/conversations/{id}") so the
// gorilla/mux matcher does not greedily bind them to the {id} route.
func registerConversationRoutes(
	r *router.Router,
	h *conversationHandler.ConversationHandler,
	createConvRateLimit coreMiddleware.Handler,
) {
	r.Post("/conversations", request.Adapt(h.CreateConversation), createConvRateLimit)
	r.Get("/conversations/me", request.Adapt(h.GetUserConversations))

	r.Get("/conversations/{id}", request.Adapt(h.GetConversationByID))
	r.Put("/conversations/{id}", request.Adapt(h.UpdateConversation))
	r.Delete("/conversations/{id}", request.Adapt(h.DeleteConversation))

	r.Post("/conversations/{id}/participants", request.Adapt(h.AddParticipants))
	r.Put("/conversations/{id}/participants/{user_id}", request.Adapt(h.UpdateParticipantRole))
	r.Delete("/conversations/{id}/participants/{user_id}", request.Adapt(h.RemoveParticipant))

	r.Get("/conversations/{id}/pinned", request.Adapt(h.GetPinnedMessages))

	r.Post("/conversations/{id}/mute", request.Adapt(h.MuteConversation))
	r.Delete("/conversations/{id}/mute", request.Adapt(h.UnmuteConversation))

	r.Get("/conversations/{id}/unread", request.Adapt(h.GetUnreadCount))

	r.Post("/conversations/{id}/typing", request.Adapt(h.SetTypingIndicator))
	r.Get("/conversations/{id}/typing", request.Adapt(h.GetTypingUsers))

	r.Put("/conversations/{id}/draft", request.Adapt(h.SaveDraft))
	r.Get("/conversations/{id}/draft", request.Adapt(h.GetDraft))
	r.Delete("/conversations/{id}/draft", request.Adapt(h.DeleteDraft))

	r.Get("/conversations/{id}/settings", request.Adapt(h.GetSettings))
	r.Put("/conversations/{id}/settings", request.Adapt(h.UpdateSettings))

	r.Post("/conversations/{id}/invites", request.Adapt(h.CreateInvite))
	r.Get("/conversations/{id}/invites", request.Adapt(h.GetConversationInvites))
	r.Delete("/conversations/{id}/invites/{invite_id}", request.Adapt(h.RevokeInvite))
}

// registerMessageRoutes wires endpoints under /conversations/{id}/messages.
//
// Literal segments (search, sync, bookmarks) come before /{id} so they
// aren't consumed by the param.
func registerMessageRoutes(r *router.Router, h *messageHandler.MessageHandler, limits messageRateLimits) {
	const base = "/conversations/{conversation_id}/messages"

	r.Post(base, request.Adapt(h.SendMessage), limits.send)
	r.Get(base, request.Adapt(h.GetMessages))

	r.Post(base+"/search", request.Adapt(h.SearchMessages), limits.search)
	r.Get(base+"/sync", request.Adapt(h.SyncMessages))
	r.Get(base+"/bookmarks", request.Adapt(h.GetBookmarks))

	r.Get(base+"/{id}", request.Adapt(h.GetMessageByID))
	r.Put(base+"/{id}", request.Adapt(h.EditMessage))
	r.Delete(base+"/{id}", request.Adapt(h.DeleteMessage))

	r.Post(base+"/{id}/reactions", request.Adapt(h.AddReaction), limits.react)
	r.Get(base+"/{id}/reactions", request.Adapt(h.GetReactions))
	r.Delete(base+"/{id}/reactions/{reaction_id}", request.Adapt(h.RemoveReaction))

	r.Post(base+"/{id}/read", request.Adapt(h.MarkAsRead))
	r.Get(base+"/{id}/delivery", request.Adapt(h.GetDeliveryStatus))

	r.Post(base+"/{id}/forward", request.Adapt(h.ForwardMessage), limits.send)

	r.Post(base+"/{id}/pin", request.Adapt(h.PinMessage))
	r.Delete(base+"/{id}/pin", request.Adapt(h.UnpinMessage))

	r.Get(base+"/{id}/thread", request.Adapt(h.GetThread))

	r.Post(base+"/{id}/bookmark", request.Adapt(h.BookmarkMessage))
	r.Delete(base+"/{id}/bookmark", request.Adapt(h.RemoveBookmark))

	r.Post(base+"/{id}/report", request.Adapt(h.ReportMessage), limits.report)
}

func registerPollRoutes(r *router.Router, h *messageHandler.MessageHandler) {
	r.Post("/polls", request.Adapt(h.CreatePoll))
	r.Post("/polls/{id}/vote", request.Adapt(h.VotePoll))
	r.Get("/polls/{id}/results", request.Adapt(h.GetPollResults))
}

func registerInviteRoutes(r *router.Router, h *conversationHandler.ConversationHandler) {
	r.Post("/invites/{code}/accept", request.Adapt(h.AcceptInvite))
}

// splitUserConvKey reverses the "userID|conversationID" composite used
// by the search rate-limit closure.
func splitUserConvKey(key string) (string, string, bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}
