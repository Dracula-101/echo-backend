package config

import (
	"fmt"
	"time"
)

func ValidateAndSetDefaults(cfg *Config) error {
	if err := validateService(&cfg.Service); err != nil {
		return err
	}

	if err := validateServer(&cfg.Server); err != nil {
		return err
	}

	if err := validateDatabase(&cfg.Database); err != nil {
		return err
	}

	if err := validateKafka(&cfg.Kafka); err != nil {
		return err
	}

	if err := validateCache(&cfg.Cache); err != nil {
		return err
	}

	if err := validateWebSocket(&cfg.WebSocket); err != nil {
		return err
	}

	if err := validateLogging(&cfg.Logging); err != nil {
		return err
	}

	if err := validateShutdown(&cfg.Shutdown); err != nil {
		return err
	}

	if err := validateMonitoring(&cfg.Monitoring); err != nil {
		return err
	}

	if err := validateSecurity(&cfg.Security); err != nil {
		return err
	}

	if err := validateFeatures(&cfg.Features); err != nil {
		return err
	}

	if err := validateLimits(&cfg.Limits); err != nil {
		return err
	}

	return nil
}

func validateService(service *ServiceConfig) error {
	if service.Name == "" {
		service.Name = "message-service"
	}

	if service.Version == "" {
		service.Version = "1.0.0"
	}

	if service.Environment == "" {
		service.Environment = "development"
	}

	return nil
}

func validateServer(server *ServerConfig) error {
	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", server.Port)
	}

	if server.Host == "" {
		server.Host = "0.0.0.0"
	}

	if server.ReadTimeout == 0 {
		server.ReadTimeout = 15 * time.Second
	}

	if server.WriteTimeout == 0 {
		server.WriteTimeout = 15 * time.Second
	}

	if server.IdleTimeout == 0 {
		server.IdleTimeout = 60 * time.Second
	}

	if server.ShutdownTimeout == 0 {
		server.ShutdownTimeout = 30 * time.Second
	}

	if server.MaxHeaderBytes == 0 {
		server.MaxHeaderBytes = 1 << 20 // 1MB
	}

	if len(server.AllowedOrigins) == 0 {
		server.AllowedOrigins = []string{"*"}
	}

	return nil
}

func validateDatabase(db *DatabaseConfig) error {
	if db.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if db.Port <= 0 || db.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", db.Port)
	}

	if db.User == "" {
		return fmt.Errorf("database user is required")
	}

	if db.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if db.SSLMode == "" {
		db.SSLMode = "disable"
	}

	if db.MaxOpenConns == 0 {
		db.MaxOpenConns = 25
	}

	if db.MaxIdleConns == 0 {
		db.MaxIdleConns = 10
	}

	if db.ConnMaxLifetime == 0 {
		db.ConnMaxLifetime = 5 * time.Minute
	}

	if db.ConnMaxIdleTime == 0 {
		db.ConnMaxIdleTime = 10 * time.Minute
	}

	return nil
}

func validateKafka(kafka *KafkaConfig) error {
	p := &kafka.Producer
	c := &kafka.Consumer

	if len(p.Brokers) == 0 && len(c.Brokers) == 0 {
		return fmt.Errorf("kafka brokers are required (producer or consumer)")
	}

	// Producer defaults
	if len(p.Brokers) > 0 {
		if p.ClientID == "" {
			p.ClientID = "message-service-producer"
		}
		if p.Topic == "" {
			p.Topic = "chat-messages"
		}
		if p.Compression == "" {
			p.Compression = "snappy"
		}
		if p.Acks == "" {
			p.Acks = "all"
		}
		if p.MaxInFlight == 0 {
			p.MaxInFlight = 5
		}
		if p.MaxRetries == 0 {
			p.MaxRetries = 5
		}
	}

	// Consumer defaults (chat messages)
	if len(c.Brokers) > 0 {
		if c.ClientID == "" {
			c.ClientID = "message-service-consumer"
		}
		if c.Topic == "" {
			c.Topic = "chat-messages"
		}
		if c.GroupID == "" {
			c.GroupID = "message-service-consumer-group"
		}
	}

	// Delivery consumer defaults — inherits brokers from main consumer if not set
	d := &kafka.DeliveryConsumer
	if len(d.Brokers) == 0 {
		d.Brokers = c.Brokers
	}
	if d.ClientID == "" {
		d.ClientID = "message-service-delivery-consumer"
	}
	if d.Topic == "" {
		d.Topic = "delivery-events"
	}
	if d.GroupID == "" {
		d.GroupID = "message-service-delivery-group"
	}
	if d.SessionTimeout == 0 {
		d.SessionTimeout = c.SessionTimeout
	}
	if d.HeartbeatInterval == 0 {
		d.HeartbeatInterval = c.HeartbeatInterval
	}

	return nil
}

func validateCache(cache *CacheConfig) error {
	if !cache.Enabled {
		return nil
	}

	if cache.Host == "" {
		return fmt.Errorf("cache host is required when cache is enabled")
	}

	if cache.Port <= 0 || cache.Port > 65535 {
		return fmt.Errorf("invalid cache port: %d", cache.Port)
	}

	if cache.PoolSize == 0 {
		cache.PoolSize = 10
	}

	if cache.MinIdleConns == 0 {
		cache.MinIdleConns = 5
	}

	if cache.MaxRetries == 0 {
		cache.MaxRetries = 3
	}

	if cache.DialTimeout == 0 {
		cache.DialTimeout = 5 * time.Second
	}

	if cache.ReadTimeout == 0 {
		cache.ReadTimeout = 3 * time.Second
	}

	if cache.WriteTimeout == 0 {
		cache.WriteTimeout = 3 * time.Second
	}

	if cache.PoolTimeout == 0 {
		cache.PoolTimeout = 4 * time.Second
	}

	if cache.IdleTimeout == 0 {
		cache.IdleTimeout = 5 * time.Minute
	}

	if cache.TTL.Message == 0 {
		cache.TTL.Message = 1 * time.Hour
	}

	if cache.TTL.Conversation == 0 {
		cache.TTL.Conversation = 30 * time.Minute
	}

	if cache.TTL.UserPresence == 0 {
		cache.TTL.UserPresence = 5 * time.Minute
	}

	return nil
}

func validateWebSocket(ws *WebSocketConfig) error {
	if ws.ReadBufferSize == 0 {
		ws.ReadBufferSize = 1024
	}

	if ws.WriteBufferSize == 0 {
		ws.WriteBufferSize = 1024
	}

	if ws.MaxMessageSize == 0 {
		ws.MaxMessageSize = 512 * 1024 // 512KB
	}

	if ws.WriteWait == 0 {
		ws.WriteWait = 10 * time.Second
	}

	if ws.PongWait == 0 {
		ws.PongWait = 60 * time.Second
	}

	if ws.PingPeriod == 0 {
		ws.PingPeriod = 54 * time.Second
	}

	if ws.MaxConnections == 0 {
		ws.MaxConnections = 10000
	}

	if ws.ClientBufferSize == 0 {
		ws.ClientBufferSize = 256
	}

	if ws.Heartbeat.Interval == 0 {
		ws.Heartbeat.Interval = 30 * time.Second
	}

	return nil
}

func validateLogging(logging *LoggingConfig) error {
	if logging.Level == "" {
		logging.Level = "info"
	}

	if logging.Format == "" {
		logging.Format = "json"
	}

	if logging.OutputPath == "" {
		logging.OutputPath = "stdout"
	}

	if logging.ErrorOutputPath == "" {
		logging.ErrorOutputPath = "stderr"
	}

	return nil
}

func validateShutdown(shutdown *ShutdownConfig) error {
	if shutdown.Timeout == 0 {
		shutdown.Timeout = 30 * time.Second
	}

	if shutdown.DrainTimeout == 0 {
		shutdown.DrainTimeout = 10 * time.Second
	}

	if shutdown.WebSocketCloseGracePeriod == 0 {
		shutdown.WebSocketCloseGracePeriod = 5 * time.Second
	}

	return nil
}

func validateMonitoring(monitoring *MonitoringConfig) error {
	if monitoring.MetricsPath == "" {
		monitoring.MetricsPath = "/metrics"
	}

	if monitoring.HealthPath == "" {
		monitoring.HealthPath = "/health"
	}

	if monitoring.TracingSampleRate <= 0 {
		monitoring.TracingSampleRate = 0.1
	}

	return nil
}

func validateSecurity(security *SecurityConfig) error {
	if security.JWTIssuer == "" {
		security.JWTIssuer = "echo-backend"
	}

	if security.JWTAudience == "" {
		security.JWTAudience = "echo-users"
	}

	if security.MaxFileSize == 0 {
		security.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}

	if security.RateLimit.RequestsPerMinute == 0 {
		security.RateLimit.RequestsPerMinute = 60
	}

	if security.RateLimit.Burst == 0 {
		security.RateLimit.Burst = 10
	}

	return nil
}

func validateFeatures(features *FeaturesConfig) error {
	return nil
}

func validateLimits(limits *LimitsConfig) error {
	if limits.MaxMessageLength == 0 {
		limits.MaxMessageLength = 4096
	}

	if limits.MaxAttachmentsPerMessage == 0 {
		limits.MaxAttachmentsPerMessage = 10
	}

	if limits.MaxMessagesPerRequest == 0 {
		limits.MaxMessagesPerRequest = 100
	}

	if limits.ConversationHistoryDays == 0 {
		limits.ConversationHistoryDays = 365
	}

	if limits.UserConversationsLimit == 0 {
		limits.UserConversationsLimit = 1000
	}

	return nil
}
