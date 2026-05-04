// Package app builds and runs the message-service. main.go is responsible
// only for env loading, logger creation, and calling app.New / app.Run.
package app

import (
	"context"

	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/server/server"
	"shared/server/shutdown"

	conversationHandler "echo-backend/services/message-service/api/v1/handler/conversation"
	messageHandler "echo-backend/services/message-service/api/v1/handler/message"
	ratelimiter "echo-backend/services/message-service/api/v1/rate_limiter"
	"echo-backend/services/message-service/internal/config"
	"echo-backend/services/message-service/internal/consumer"
	msgPublisher "echo-backend/services/message-service/internal/messaging"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service"
	cacheSvc "echo-backend/services/message-service/internal/service/cache"
	"echo-backend/services/message-service/internal/service/idempotency"
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

	deliveryConsumer, err := newKafkaConsumer(cfg.Kafka.DeliveryConsumer, log)
	if err != nil {
		return nil, err
	}

	messageCache := cacheSvc.NewMessageCache(cacheClient, log)

	messageRepo := repo.NewMessageRepository(db, cacheClient, log)
	deliveryRepo := repo.NewDeliveryRepository(db, cacheClient, log)
	conversationRepo := repo.NewConversationRepository(db, cacheClient, log)
	userRepo := repo.NewUserRepository(db)
	reactionRepo := repo.NewReactionRepository(db, cacheClient, log)
	pollRepo := repo.NewPollRepository(db, cacheClient, log)
	typingRepo := repo.NewTypingRepository(db, log)
	draftRepo := repo.NewDraftRepository(db, log)
	bookmarkRepo := repo.NewBookmarkRepository(db, log)
	settingsRepo := repo.NewConversationSettingsRepository(db, log)
	inviteRepo := repo.NewInviteRepository(db, log)
	reportRepo := repo.NewReportRepository(db, log)
	_ = repo.NewOutboxRepository(log) // wired into send/edit/delete paths in subsequent steps

	idempotencyMgr := idempotency.NewManager(cacheClient, log)

	publisher := msgPublisher.NewKafkaEventPublisher(producer, cfg.Kafka.DeliveryConsumer.Topic, log)

	messageService := service.NewMessageService(messageRepo, deliveryRepo, conversationRepo, userRepo, publisher, messageCache, log)
	conversationService := service.NewConversationService(conversationRepo, userRepo, publisher, messageCache, log)
	deliveryService := service.NewDeliveryService(deliveryRepo, log)
	reactionService := service.NewReactionService(reactionRepo, messageRepo, conversationRepo, publisher, log)
	pollService := service.NewPollService(pollRepo, messageRepo, conversationRepo, publisher, log)
	typingService := service.NewTypingService(typingRepo, publisher, log)
	draftService := service.NewDraftService(draftRepo, log)
	bookmarkService := service.NewBookmarkService(bookmarkRepo, log)
	settingsService := service.NewConversationSettingsService(settingsRepo, log)
	inviteService := service.NewInviteService(inviteRepo, conversationRepo, log)
	reportService := service.NewReportService(reportRepo, log)

	mh := messageHandler.NewMessageHandler(messageService, reactionService, pollService, bookmarkService, reportService, idempotencyMgr, log)
	ch := conversationHandler.NewConversationHandler(conversationService, messageService, typingService, draftService, settingsService, inviteService, log)
	deliveryEventConsumer := consumer.NewDeliveryEventConsumer(deliveryService, log)

	rl := ratelimiter.NewRateLimiter(cacheClient, log)

	srv, err := buildHTTPServer(cfg, mh, ch, rl, db, cacheClient, log)
	if err != nil {
		return nil, err
	}
	a.server = srv

	deliveryCtx, deliveryCancel := context.WithCancel(context.Background())
	go startTopicConsumer(deliveryCtx, deliveryConsumer, deliveryEventConsumer, cfg.Kafka.DeliveryConsumer.Topic, log)

	a.shutdownMgr = buildShutdown(shutdownDeps{
		cfg: cfg,
		log: log,
		srv: srv,
		consumers: []consumerHandle{
			{name: "delivery-events", topic: cfg.Kafka.DeliveryConsumer.Topic, cancel: deliveryCancel, consumer: deliveryConsumer},
		},
		producer:    producer,
		db:          db,
		cacheClient: cacheClient,
	})

	return a, nil
}

func (a *App) Run() error {
	a.log.Info("Starting Message Service",
		logger.String("address", a.server.Address()),
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
		a.log.Info("Message Service stopped gracefully")
	}
	return nil
}
