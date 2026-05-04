package app

import (
	"fmt"

	"shared/pkg/cache"
	cacheRedis "shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	"shared/pkg/storage/r2"
	env "shared/server/env"

	"media-service/internal/config"
	"media-service/internal/service"
)

func newDatabase(cfg config.DatabaseConfig, log logger.Logger) (database.Database, error) {
	log.Debug("Creating Postgres client",
		logger.String("host", cfg.Postgres.Host),
		logger.Int("port", cfg.Postgres.Port),
		logger.String("database", cfg.Postgres.DBName),
	)
	return postgres.New(database.Config{
		Host:            cfg.Postgres.Host,
		Port:            cfg.Postgres.Port,
		User:            cfg.Postgres.User,
		Password:        cfg.Postgres.Password,
		Database:        cfg.Postgres.DBName,
		SSLMode:         cfg.Postgres.SSLMode,
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Postgres.ConnMaxIdleTime,
	})
}

func newCache(cfg config.CacheConfig, log logger.Logger) (cache.Cache, error) {
	log.Debug("Creating Redis cache client",
		logger.String("host", cfg.RedisConfig.RedisHost),
		logger.Int("port", cfg.RedisConfig.RedisPort),
		logger.Int("db", cfg.RedisConfig.RedisDB),
	)
	return cacheRedis.New(cache.Config{
		Host:         cfg.RedisConfig.RedisHost,
		Port:         cfg.RedisConfig.RedisPort,
		Password:     cfg.RedisConfig.RedisPassword,
		DB:           cfg.RedisConfig.RedisDB,
		DialTimeout:  cfg.RedisConfig.RedisDialTimeout,
		PoolSize:     cfg.RedisConfig.RedisPoolSize,
		MinIdleConns: cfg.RedisConfig.RedisMinIdleConns,
	})
}

func newKafkaProducer(p config.KafkaProducerConfig, log logger.Logger) (messaging.Producer, error) {
	log.Debug("Creating Kafka producer",
		logger.Strings("brokers", p.Brokers),
		logger.String("topic", p.Topic),
	)
	return kafka.NewProducer(messaging.ProducerConfig{
		Brokers:           p.Brokers,
		ClientID:          p.ClientID,
		MaxRetries:        p.MaxRetries,
		RetryBackoff:      p.RetryBackoff,
		Compression:       p.Compression,
		Acks:              p.Acks,
		EnableIdempotence: p.EnableIdempotence,
		MaxInFlight:       p.MaxInFlight,
		DialTimeout:       p.DialTimeout,
	})
}

// newStorageProvider routes on the configured provider name. Only R2 is
// wired today; adding a new backend is one switch case.
func newStorageProvider(cfg *config.Config, log logger.Logger) (service.StorageProvider, error) {
	log.Debug("Creating storage provider",
		logger.String("provider", cfg.Storage.Provider),
		logger.String("bucket", cfg.Storage.Bucket),
	)
	switch cfg.Storage.Provider {
	case "r2":
		return r2.New(r2.Config{
			AccountID:       env.GetEnv("R2_ACCOUNT_ID"),
			AccessKeyID:     cfg.Storage.AccessKeyID,
			SecretAccessKey: cfg.Storage.SecretAccessKey,
			Bucket:          cfg.Storage.Bucket,
			PublicURL:       cfg.Storage.PublicURL,
			CDNBaseURL:      cfg.Storage.CDNBaseURL,
			UseCDN:          cfg.Storage.UseCDN,
			ImageQuality:    cfg.Features.ImageProcessing.Quality,
		}, log)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.Storage.Provider)
	}
}
