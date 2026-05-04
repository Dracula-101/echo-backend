package app

import (
	"context"
	"fmt"

	email "shared/pkg/email"
	mailgun "shared/pkg/email/mailgun"

	"shared/pkg/cache"
	"shared/pkg/cache/memory"
	cacheRedis "shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	"shared/pkg/windowstore"
	redisWindowStore "shared/pkg/windowstore/redis"
	"shared/server/common/hashing"
	"shared/server/common/token"

	"auth-service/internal/config"
	"auth-service/internal/firebase"
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

func newMemCache(log logger.Logger) cache.Cache {
	log.Info("Initializing in-memory cache")
	return memory.NewMemCache()
}

func newWindowStore(cfg *config.Config, log logger.Logger) windowstore.WindowStore {
	log.Info("Initializing window store for rate limiting")
	return redisWindowStore.NewRedis(windowstore.Config{
		Addr:     fmt.Sprintf("%s:%d", cfg.Cache.RedisConfig.RedisHost, cfg.Cache.RedisConfig.RedisPort),
		DB:       cfg.Cache.RedisConfig.RedisDB,
		Password: cfg.Cache.RedisConfig.RedisPassword,
	}, log)
}

func newKafkaProducer(cfg config.KafkaProducerConfig, log logger.Logger) (messaging.Producer, error) {
	log.Debug("Creating Kafka producer",
		logger.Strings("brokers", cfg.Brokers),
		logger.String("topic", cfg.Topic),
	)
	return kafka.NewProducer(messaging.ProducerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		MaxRetries:        cfg.MaxRetries,
		RetryBackoff:      cfg.RetryBackoff,
		Compression:       cfg.Compression,
		Acks:              cfg.Acks,
		EnableIdempotence: cfg.EnableIdempotence,
		MaxInFlight:       cfg.MaxInFlight,
		DialTimeout:       cfg.DialTimeout,
	})
}

// newEmailService routes on the configured provider name. Returns a
// pointer with a nil interface inside when no provider is configured —
// callers should treat email as best-effort in that mode.
func newEmailService(cfg *config.Config, log logger.Logger) *email.EmailService {
	log.Debug("Setting up Email service", logger.String("provider", cfg.Email.Provider))
	var svc email.EmailService
	switch cfg.Email.Provider {
	case "mailgun":
		svc = mailgun.New(mailgun.MailgunConfig{
			Domain:  cfg.Email.Mailgun.Domain,
			APIKey:  cfg.Email.Mailgun.APIKey,
			BaseURL: cfg.Email.Mailgun.BaseURL,
		})
		log.Info("Mailgun email service initialized")
	default:
		log.Warn("No valid email provider configured; email functionality disabled")
	}
	return &svc
}

func newTokenService(cfg *config.Config) (*token.JWTTokenService, error) {
	keyset, err := token.NewStaticKeySet([]byte(cfg.Auth.JWT.SecretKey))
	if err != nil {
		return nil, err
	}
	return token.NewJWTTokenService(token.Config{
		KeySet:          keyset,
		Issuer:          cfg.Auth.JWT.Issuer,
		Audience:        []string{cfg.Auth.JWT.Audience},
		AccessTokenTTL:  cfg.Auth.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.Auth.JWT.RefreshTokenTTL,
		Leeway:          cfg.Auth.JWT.Leeway,
	})
}

// newHashingService applies sane fixed parameters for values not yet
// surfaced in config (memory, threads, scrypt N/R/P). Move them into
// config when password-policy changes are considered.
func newHashingService(cfg *config.Config) (*hashing.HashingService, error) {
	return hashing.NewService(hashing.Config{
		Default: hashing.Algorithm(cfg.Auth.Hash.Default),
		Argon2: hashing.Argon2Config{
			SaltLength: uint32(cfg.Auth.Hash.SaltLength),
			Time:       uint32(cfg.Auth.Hash.Iterations),
			Memory:     uint32(64 * 1024),
			Threads:    uint8(4),
			KeyLength:  uint32(cfg.Auth.Hash.KeyLength),
		},
		Bcrypt: hashing.BcryptConfig{Cost: cfg.Auth.Hash.Cost},
		Scrypt: hashing.ScryptConfig{
			SaltLength: cfg.Auth.Hash.SaltLength,
			N:          1 << uint8(cfg.Auth.Hash.Iterations),
			R:          8,
			P:          1,
			KeyLength:  cfg.Auth.Hash.KeyLength,
		},
	})
}

// newFirebaseClient is best-effort: when no credentials file is
// configured, returns nil and the caller treats Firebase features
// (currently phone login) as unavailable.
func newFirebaseClient(cfg *config.Config, log logger.Logger) (*firebase.Client, error) {
	if cfg.Firebase.CredentialsFile == "" {
		log.Warn("Firebase credentials not configured; phone login unavailable")
		return nil, nil
	}
	return firebase.New(context.Background(), firebase.Config{
		CredentialsFile: cfg.Firebase.CredentialsFile,
	})
}
