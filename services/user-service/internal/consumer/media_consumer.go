package consumer

import (
	"context"
	"encoding/json"
	"shared/pkg/logger"
	"shared/pkg/messaging"
	"time"
	"user-service/internal/config"
	"user-service/internal/errors"
	repository "user-service/internal/repo"

	pkgErrors "shared/pkg/errors"
)

type MediaEventsConsumer interface {
	Start(ctx context.Context) pkgErrors.AppError
	Close() pkgErrors.AppError
}

type mediaEventConsumer struct {
	consumer messaging.Consumer
	cfg      config.KafkaConsumerConfig
	userRepo repository.UserRepository
	log      logger.Logger
}

func NewMediaEventConsumer(
	consumer messaging.Consumer,
	cfg config.KafkaConsumerConfig,
	userRepo repository.UserRepository,
	log logger.Logger,
) MediaEventsConsumer {
	return &mediaEventConsumer{
		consumer: consumer,
		cfg:      cfg,
		userRepo: userRepo,
		log:      log,
	}
}

func (m *mediaEventConsumer) Start(ctx context.Context) pkgErrors.AppError {
	handler := &mediaMessageHandler{
		userRepo: m.userRepo,
		log:      m.log,
	}
	return pkgErrors.FromError(m.consumer.Consume(ctx, []string{m.cfg.Topic}, handler), errors.ErrCodeConsumerStartFailed, "failed to start media event consumer")
}

func (m *mediaEventConsumer) Close() pkgErrors.AppError {
	return pkgErrors.FromError(m.consumer.Close(), errors.ErrCodeConsumerCloseFailed, "failed to close media event consumer")
}

type mediaMessageHandler struct {
	userRepo repository.UserRepository
	log      logger.Logger
}

type mediaEventBase struct {
	EventType string          `json:"event_type"`
	FileId    string          `json:"file_id"`
	UserId    string          `json:"user_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"-"`
}

type ProfilePhotoUploadedEvent struct {
	UserId        string     `json:"user_id"`
	FileId        string     `json:"file_id"`
	Timestamp     time.Time  `json:"timestamp"`
	ThumbnailURLs Thumbnails `json:"thumbnail"`
}

type Thumbnails struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Large  string `json:"large"`
}

func (h *mediaMessageHandler) Handle(ctx context.Context, message *messaging.Message) error {
	h.log.Debug("Processing media event message",
		logger.String("topic", message.Topic),
		logger.String("key", string(message.Key)),
	)

	var base mediaEventBase
	if err := json.Unmarshal(message.Value, &base); err != nil {
		h.log.Error("Failed to unmarshal media event base",
			logger.String("value", string(message.Value)),
			logger.Error(err),
		)
		return nil
	}

	switch base.EventType {
	case "profile_photo_uploaded":
		return h.handleProfilePhotoUploaded(ctx, message.Value, base.UserId, base.FileId)
	default:
		h.log.Debug("Skipping unknown event type",
			logger.String("event_type", base.EventType),
		)
	}

	return nil
}

func (h *mediaMessageHandler) handleProfilePhotoUploaded(ctx context.Context, raw []byte, userId, fileId string) error {
	h.log.Info("Handling profile photo uploaded event",
		logger.String("user_id", userId),
		logger.String("file_id", fileId),
	)

	var event ProfilePhotoUploadedEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		h.log.Error("Failed to unmarshal profile photo uploaded event",
			logger.String("user_id", userId),
			logger.Error(err),
		)
		return nil
	}

	if _, err := h.userRepo.UpdateProfile(ctx, event.UserId, repository.UpdateProfileParams{
		AvatarThumbnailURL: &event.ThumbnailURLs.Small,
	}); err != nil {
		h.log.Error("Failed to update user profile with new thumbnail URL",
			logger.String("user_id", event.UserId),
			logger.String("thumbnail_url", event.ThumbnailURLs.Small),
			logger.Error(err),
		)
		return nil
	}

	h.log.Info("Successfully processed profile photo uploaded event",
		logger.String("user_id", event.UserId),
		logger.String("file_id", event.FileId),
	)

	return nil
}
