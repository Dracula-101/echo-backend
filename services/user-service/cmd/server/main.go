package main

import (
	"fmt"

	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	env "shared/server/env"

	"user-service/internal/app"
	"user-service/internal/config"
)

func main() {
	env.LoadEnv()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	log := mustCreateLogger(cfg.Service.Name)
	defer log.Sync()

	log.Info("Initializing application",
		logger.String("service", cfg.Service.Name),
		logger.String("version", cfg.Service.Version),
	)

	a, err := app.New(cfg, log)
	if err != nil {
		log.Fatal("Failed to build application", logger.Error(err))
	}

	if err := a.Run(); err != nil {
		log.Fatal("Server error", logger.Error(err))
	}
}

func mustCreateLogger(name string) logger.Logger {
	log, err := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Format:     logger.GetLoggerFormat(),
		Output:     logger.GetLoggerOutput(),
		TimeFormat: logger.GetLoggerTimeFormat(),
		Service:    name,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	return log
}
