package app

import (
	"context"

	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/server/server"
	"shared/server/shutdown"

	"user-service/internal/config"
)

type shutdownDeps struct {
	cfg         *config.Config
	log         logger.Logger
	srv         *server.Server
	db          database.Database
	cacheClient cache.Cache
}

func buildShutdown(deps shutdownDeps) *shutdown.Manager {
	mgr := shutdown.New(
		shutdown.WithTimeout(deps.cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(deps.log),
	)

	if deps.srv != nil {
		mgr.RegisterWithPriority(
			"http-server",
			shutdown.ServerShutdownHook(deps.srv),
			shutdown.PriorityHigh,
		)
	}

	if deps.cfg.Shutdown.WaitForConnections && deps.cfg.Shutdown.DrainTimeout > 0 {
		mgr.RegisterWithOptions(
			"drain-connections",
			shutdown.DelayHook(deps.cfg.Shutdown.DrainTimeout),
			shutdown.PriorityHigh,
			deps.cfg.Shutdown.DrainTimeout,
		)
	}

	if deps.cacheClient != nil {
		mgr.RegisterWithPriority(
			"cache",
			shutdown.Hook(func(ctx context.Context) error {
				deps.log.Info("Closing cache connection")
				return deps.cacheClient.Close()
			}),
			shutdown.PriorityNormal,
		)
	}

	if deps.db != nil {
		mgr.RegisterWithPriority(
			"database",
			shutdown.Hook(func(ctx context.Context) error {
				deps.log.Info("Closing database connection")
				return deps.db.Close()
			}),
			shutdown.PriorityNormal,
		)
	}

	mgr.RegisterWithPriority(
		"logger-sync",
		shutdown.Hook(func(ctx context.Context) error {
			deps.log.Info("Syncing logger before shutdown")
			return deps.log.Sync()
		}),
		shutdown.PriorityLow,
	)

	return mgr
}
