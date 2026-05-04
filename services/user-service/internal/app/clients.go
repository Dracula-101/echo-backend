package app

import (
	"context"

	"shared/pkg/cache"
	cacheRedis "shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	searchClient "shared/pkg/search"
	"shared/pkg/search/meilisearch"
	"shared/server/common/token"

	"user-service/internal/config"
	"user-service/internal/consumer"
	repository "user-service/internal/repo"
)

func newDatabase(cfg config.DatabaseConfig, log logger.Logger) (database.Database, error) {
	log.Debug("Creating Postgres client",
		logger.String("host", cfg.Postgres.Host),
		logger.Int("port", cfg.Postgres.Port),
		logger.String("user", cfg.Postgres.User),
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

func newSearchClient(cfg *config.Config, log logger.Logger) (searchClient.Search, error) {
	log.Debug("Creating Meilisearch client",
		logger.String("host", cfg.Search.Meilisearch.Host),
	)
	return meilisearch.New(meilisearch.Config{
		Host:    cfg.Search.Meilisearch.Host,
		APIKey:  cfg.Search.Meilisearch.APIKey,
		Timeout: cfg.Search.Meilisearch.Timeout,
	}, log)
}

// startMediaConsumer creates the Kafka consumer for media-events and
// launches its read goroutine. The MediaEventsConsumer Start runs until
// its context is cancelled.
func startMediaConsumer(cfg *config.Config, userRepo repository.UserRepository, log logger.Logger) (*consumer.MediaEventsConsumer, error) {
	c := cfg.Kafka.Consumer
	log.Debug("Creating messaging consumer",
		logger.Strings("brokers", c.Brokers),
		logger.String("topic", c.Topic),
		logger.String("groupID", c.GroupID),
	)
	kafkaConsumer, err := kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           c.Brokers,
		ClientID:          c.ClientID,
		GroupID:           c.GroupID,
		SessionTimeout:    c.SessionTimeout,
		HeartbeatInterval: c.HeartbeatInterval,
	})
	if err != nil {
		return nil, err
	}
	mediaConsumer := consumer.NewMediaEventConsumer(kafkaConsumer, c, userRepo, log)
	log.Info("Messaging consumer created")

	go func() {
		if err := mediaConsumer.Start(context.Background()); err != nil {
			log.Error("Failed to start media event consumer", logger.Error(err))
		} else {
			log.Info("Media event consumer started")
		}
	}()
	return &mediaConsumer, nil
}

func newTokenService(cfg *config.Config) (*token.JWTTokenService, error) {
	keyset, err := token.NewStaticKeySet([]byte(cfg.JWT.SecretKey))
	if err != nil {
		return nil, err
	}
	return token.NewJWTTokenService(token.Config{
		KeySet:          keyset,
		Issuer:          cfg.JWT.Issuer,
		Audience:        []string{cfg.JWT.Audience},
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
}
