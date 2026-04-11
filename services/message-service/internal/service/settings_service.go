package service

import (
	"context"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"time"

	"github.com/google/uuid"
)

// ConversationSettingsResponse is the service-layer representation of conversation settings.
type ConversationSettingsResponse struct {
	ID                           string    `json:"id"`
	ConversationID               string    `json:"conversation_id"`
	DisappearingMessagesEnabled  bool      `json:"disappearing_messages_enabled"`
	DisappearingMessagesDuration *int      `json:"disappearing_messages_duration"`
	ReadReceiptsEnabled          bool      `json:"read_receipts_enabled"`
	TypingIndicatorsEnabled      bool      `json:"typing_indicators_enabled"`
	LinkPreviewsEnabled          bool      `json:"link_previews_enabled"`
	AutoDownloadMedia            bool      `json:"auto_download_media"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

// ConversationSettingsServiceInterface defines methods for managing conversation settings.
type ConversationSettingsServiceInterface interface {
	GetSettings(ctx context.Context, conversationID uuid.UUID) (*ConversationSettingsResponse, *msgError.MessageError)
	UpdateSettings(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, input repo.ConversationSettingsInput) *msgError.MessageError
}

type conversationSettingsService struct {
	settingsRepo repo.ConversationSettingsRepository
	logger       logger.Logger
}

func NewConversationSettingsService(settingsRepo repo.ConversationSettingsRepository, log logger.Logger) ConversationSettingsServiceInterface {
	return &conversationSettingsService{
		settingsRepo: settingsRepo,
		logger:       log,
	}
}

func (s *conversationSettingsService) GetSettings(ctx context.Context, conversationID uuid.UUID) (*ConversationSettingsResponse, *msgError.MessageError) {
	settings, err := s.settingsRepo.GetSettings(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to get conversation settings",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to get conversation settings",
			Code:    msgError.CodeConversationFetchFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to get conversation settings"),
		}
	}

	if settings == nil {
		return nil, nil
	}

	return &ConversationSettingsResponse{
		ID:                           settings.ID.String(),
		ConversationID:               settings.ConversationID.String(),
		DisappearingMessagesEnabled:  settings.DisappearingMessagesEnabled,
		DisappearingMessagesDuration: settings.DisappearingMessagesDuration,
		ReadReceiptsEnabled:          settings.ReadReceiptsEnabled,
		TypingIndicatorsEnabled:      settings.TypingIndicatorsEnabled,
		LinkPreviewsEnabled:          settings.LinkPreviewsEnabled,
		AutoDownloadMedia:            settings.AutoDownloadMedia,
		CreatedAt:                    settings.CreatedAt,
		UpdatedAt:                    settings.UpdatedAt,
	}, nil
}

func (s *conversationSettingsService) UpdateSettings(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, input repo.ConversationSettingsInput) *msgError.MessageError {
	err := s.settingsRepo.UpdateSettings(ctx, conversationID, input)
	if err != nil {
		s.logger.Error("Failed to update conversation settings",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return &msgError.MessageError{
			Message: "Failed to update conversation settings",
			Code:    msgError.CodeConversationUpdateFailed,
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to update conversation settings"),
		}
	}

	return nil
}
