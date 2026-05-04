// Package app builds and runs the auth-service. main.go is responsible
// only for env loading, logger creation, and calling app.New / app.Run.
package app

import (
	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/pkg/windowstore"
	"shared/server/server"
	"shared/server/shutdown"

	"auth-service/api/v1/handler"
	ratelimiter "auth-service/api/v1/rate_limiter"
	"auth-service/internal/config"
	repository "auth-service/internal/repo"
	"auth-service/internal/repo/email_verification"
	"auth-service/internal/repo/login_history"
	"auth-service/internal/repo/outbox"
	"auth-service/internal/repo/password_reset"
	"auth-service/internal/repo/session"
	"auth-service/internal/service"
	authCacheSvc "auth-service/internal/service/cache"
	"auth-service/internal/service/location"
	sessionSvc "auth-service/internal/service/session"
	"auth-service/internal/service/sms"
)

type App struct {
	cfg *config.Config
	log logger.Logger

	server      *server.Server
	shutdownMgr *shutdown.Manager
}

func New(cfg *config.Config, log logger.Logger) (*App, error) {
	a := &App{cfg: cfg, log: log}

	db, err := newDatabase(cfg.Database, log)
	if err != nil {
		return nil, err
	}
	log.Info("Database client created")

	var (
		cacheClient cache.Cache
		memCache    cache.Cache
		winStore    windowstore.WindowStore
	)
	if cfg.Cache.Enabled {
		cacheClient, err = newCache(cfg.Cache, log)
		if err != nil {
			return nil, err
		}
		log.Info("Cache client created")
		memCache = newMemCache(log)
		winStore = newWindowStore(cfg, log)
	} else {
		log.Info("Cache is disabled in configuration")
	}

	producer, err := newKafkaProducer(cfg.Kafka.Producer, log)
	if err != nil {
		return nil, err
	}

	firebaseClient, err := newFirebaseClient(cfg, log)
	if err != nil {
		return nil, err
	}
	if firebaseClient != nil {
		log.Info("Firebase client initialized")
	}

	authCache := authCacheSvc.NewAuthCache(cacheClient, log)
	sessionCache := sessionSvc.NewSessionCache(cacheClient, log)
	rl := ratelimiter.NewRateLimiter(cacheClient, winStore, log)

	emailService := newEmailService(cfg, log)

	tokenService, err := newTokenService(cfg)
	if err != nil {
		return nil, err
	}
	log.Info("Token service created")

	hashingService, err := newHashingService(cfg)
	if err != nil {
		return nil, err
	}
	log.Info("Hashing service created")

	locationService := location.NewLocationService(cfg.LocationService.Endpoint, log)
	log.Info("Sms service configuration",
		logger.String("provider", cfg.Sms.Provider),
		logger.Bool("enabled", cfg.Sms.Enabled),
	)
	smsService := sms.NewSmsService(cfg.Sms.Twilio.AccountSID, cfg.Sms.Twilio.MessagingServiceSID, cfg.Sms.Twilio.AuthToken, cfg.Sms.Twilio.FromNumber, log)

	_ = outbox.NewOutboxRepository(db, log)
	loginHistoryRepo := login_history.NewLoginHistoryRepo(db, log)
	sessionRepo := session.NewSessionRepo(db, log)
	sessionService := sessionSvc.NewSessionService(sessionRepo, sessionCache, *tokenService, log, cfg.Cache)
	emailVerificationRepo := email_verification.NewEmailVerificationTokenRepo(db, log)
	resetTokenRepo := password_reset.NewPasswordResetTokenRepo(db, log)
	authRepo := repository.NewAuthRepository(db, log)

	authService := service.NewAuthServiceBuilder().
		WithRepo(authRepo).
		WithLoginHistoryRepo(loginHistoryRepo).
		WithEmailVerificationRepo(emailVerificationRepo).
		WithPasswordResetRepo(resetTokenRepo).
		WithSessionRepo(sessionRepo).
		WithAuthCache(authCache).
		WithSessionCache(sessionCache).
		WithRateLimiter(rl).
		WithSmsService(smsService).
		WithTokenService(*tokenService).
		WithHashingService(*hashingService).
		WithEmailService(*emailService).
		WithConfig(&cfg.Auth).
		WithLogger(log).
		Build()

	authHandler := handler.NewAuthHandler(authService, sessionService, *locationService, authCache, rl, firebaseClient, log)

	srv, err := buildHTTPServer(cfg, authHandler, memCache, *locationService, sessionService, sessionCache, rl, db, cacheClient, log)
	if err != nil {
		return nil, err
	}
	a.server = srv

	a.shutdownMgr = buildShutdown(shutdownDeps{
		cfg:         cfg,
		log:         log,
		srv:         srv,
		producer:    producer,
		winStore:    winStore,
		memCache:    memCache,
		cacheClient: cacheClient,
		db:          db,
	})

	return a, nil
}

func (a *App) Run() error {
	a.log.Info("Starting Auth Service",
		logger.String("host", a.cfg.Server.Host),
		logger.Int("port", a.cfg.Server.Port),
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- a.server.Start()
	}()

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		_ = a.shutdownMgr.Wait()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !server.IsServerDownErr(err) {
			return err
		}
		a.log.Info("Server stopped")
	case <-shutdownDone:
		a.log.Info("Auth Service stopped gracefully")
	}
	return nil
}
