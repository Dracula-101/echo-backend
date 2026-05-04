// Package app builds and runs the media-service. main.go is responsible
// only for env loading, logger creation, and calling app.New / app.Run.
package app

import (
	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/pkg/media"
	"shared/server/server"
	"shared/server/shutdown"

	"media-service/api/v1/handler"
	"media-service/internal/config"
	repository "media-service/internal/repo"
	"media-service/internal/service"
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

	producer, err := newKafkaProducer(cfg.Kafka.Producer, log)
	if err != nil {
		return nil, err
	}

	storageProvider, err := newStorageProvider(cfg, log)
	if err != nil {
		return nil, err
	}

	fileRepo := repository.NewFileRepository(db, log)
	mediaService := service.NewMediaServiceBuilder().
		WithFileRepo(fileRepo).
		WithCache(cacheClient).
		WithConfig(cfg).
		WithLogger(log).
		WithStorageProvider(storageProvider).
		WithProducer(producer).
		Build()

	mediaProcessor := media.NewProcessor()
	mediaHandler := handler.NewHandler(mediaService, mediaProcessor, cfg, log)

	srv, err := buildHTTPServer(cfg, mediaHandler, db, cacheClient, log)
	if err != nil {
		return nil, err
	}
	a.server = srv

	a.shutdownMgr = buildShutdown(shutdownDeps{
		cfg:         cfg,
		log:         log,
		srv:         srv,
		producer:    producer,
		db:          db,
		cacheClient: cacheClient,
	})

	return a, nil
}

func (a *App) Run() error {
	a.log.Info("Starting Media Service",
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
		a.log.Info("Media Service stopped gracefully")
	}
	return nil
}
