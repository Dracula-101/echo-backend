// Package app builds and runs the user-service. main.go is responsible
// only for env loading, logger creation, and calling app.New / app.Run.
package app

import (
	"context"

	"shared/pkg/cache"
	"shared/pkg/cache/memory"
	"shared/pkg/logger"
	windowStore "shared/pkg/windowstore/memory"
	"shared/server/server"
	"shared/server/shutdown"

	"user-service/api/v1/handler"
	ratelimiter "user-service/api/v1/rate_limiter"
	"user-service/internal/config"
	repository "user-service/internal/repo"
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/contacts"
	"user-service/internal/repo/devices"
	"user-service/internal/repo/privacy_overrides"
	"user-service/internal/repo/settings"
	"user-service/internal/service"
	userCacheSvc "user-service/internal/service/cache"
	"user-service/internal/service/contact"
	"user-service/internal/service/location"
	"user-service/internal/service/search"
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

	var cacheClient cache.Cache
	if cfg.Cache.Enabled {
		cacheClient, err = newCache(cfg.Cache, log)
		if err != nil {
			return nil, err
		}
		log.Info("Cache client created")
	} else {
		log.Info("Cache is disabled in configuration")
	}

	searchClient, err := newSearchClient(cfg, log)
	if err != nil {
		return nil, err
	}
	log.Info("Search client created")

	tokenService, err := newTokenService(cfg)
	if err != nil {
		return nil, err
	}

	memCache := memory.NewMemCache()
	userCache := userCacheSvc.NewUserCache(cacheClient, log)
	userRepo := repository.NewUserRepository(db, log)
	blockedRepo := blocked_users.NewBlockedRepository(db, log)
	contactsRepo := contacts.NewContactRepository(db, log)
	settingsRepo := settings.NewSettingsRepository(db, log)
	privacyRepo := privacy_overrides.NewPrivacyOverrideRepository(db, log)
	deviceRepo := devices.NewDeviceRepository(db, log)

	if _, err := startMediaConsumer(cfg, userRepo, log); err != nil {
		return nil, err
	}

	rl := ratelimiter.NewRateLimiter(cacheClient, windowStore.NewMemory(log), log)

	locationService := location.NewLocationService(cfg.Server.LocationServiceEndpoint, log)
	userService := service.NewUserService(userRepo, blockedRepo, contactsRepo, settingsRepo, privacyRepo, userCache, log)
	searchService := search.NewSearchService(userRepo, searchClient, userCache, log)
	go searchService.CreateIndex(context.Background())
	contactService := contact.NewContactService(contactsRepo, userRepo, blockedRepo, userCache, log)
	userHandler := handler.NewUserHandler(userService, searchService, contactService, locationService, blockedRepo, settingsRepo, deviceRepo, userCache, tokenService, log)

	srv, err := buildHTTPServer(cfg, userHandler, locationService, memCache, rl, db, cacheClient, log)
	if err != nil {
		return nil, err
	}
	a.server = srv

	a.shutdownMgr = buildShutdown(shutdownDeps{
		cfg:         cfg,
		log:         log,
		srv:         srv,
		db:          db,
		cacheClient: cacheClient,
	})

	return a, nil
}

func (a *App) Run() error {
	a.log.Info("Starting User Service",
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
		a.log.Info("User Service stopped gracefully")
	}
	return nil
}
