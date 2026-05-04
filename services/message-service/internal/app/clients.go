package app

import (
	"shared/pkg/cache"
	cacheRedis "shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"

	"echo-backend/services/message-service/internal/config"
)

func newDatabase(cfg config.DatabaseConfig, log logger.Logger) (database.Database, error) {
	log.Debug("Creating Postgres client",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("database", cfg.DBName),
	)
	return postgres.New(database.Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        cfg.DBName,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})
}

func newCache(cfg config.CacheConfig, log logger.Logger) (cache.Cache, error) {
	log.Debug("Creating Redis cache client",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.Int("db", cfg.DB),
	)
	return cacheRedis.New(cache.Config{
		Host:         cfg.Host,
		Port:         cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
}

func newKafkaProducer(cfg config.KafkaProducerConfig, log logger.Logger) (messaging.Producer, error) {
	log.Debug("Creating Kafka producer",
		logger.Strings("brokers", cfg.Brokers),
		logger.String("topic", cfg.Topic),
	)
	return kafka.NewProducer(messaging.ProducerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		Compression:       cfg.Compression,
		BatchSize:         cfg.BatchSize,
		LingerMs:          cfg.LingerMs,
		Acks:              cfg.Acks,
		EnableIdempotence: cfg.EnableIdempotence,
		MaxInFlight:       cfg.MaxInFlight,
		MaxRetries:        cfg.MaxRetries,
		RetryBackoff:      cfg.RetryBackoff,
		DialTimeout:       cfg.DialTimeout,
	})
}

// newKafkaConsumer creates a Kafka consumer using the supplied config.
// Reused for any consumer group; the topic is passed at Consume time.
func newKafkaConsumer(cfg config.KafkaConsumerConfig, log logger.Logger) (messaging.Consumer, error) {
	log.Debug("Creating Kafka consumer",
		logger.Strings("brokers", cfg.Brokers),
		logger.String("group_id", cfg.GroupID),
	)
	return kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		SessionTimeout:    cfg.SessionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
	})
}
