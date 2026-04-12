package repo

import (
	"context"
	"fmt"
	"shared/pkg/database"
	"shared/pkg/logger"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConversationSettings represents a row in messages.conversation_settings.
type ConversationSettings struct {
	ID                           uuid.UUID `json:"id"`
	ConversationID               uuid.UUID `json:"conversation_id"`
	DisappearingMessagesEnabled  bool      `json:"disappearing_messages_enabled"`
	DisappearingMessagesDuration *int      `json:"disappearing_messages_duration"`
	ReadReceiptsEnabled          bool      `json:"read_receipts_enabled"`
	TypingIndicatorsEnabled      bool      `json:"typing_indicators_enabled"`
	LinkPreviewsEnabled          bool      `json:"link_previews_enabled"`
	AutoDownloadMedia            bool      `json:"auto_download_media"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

// ConversationSettingsInput holds optional fields for updating conversation settings.
type ConversationSettingsInput struct {
	DisappearingMessagesEnabled  *bool `json:"disappearing_messages_enabled,omitempty"`
	DisappearingMessagesDuration *int  `json:"disappearing_messages_duration,omitempty"`
	ReadReceiptsEnabled          *bool `json:"read_receipts_enabled,omitempty"`
	TypingIndicatorsEnabled      *bool `json:"typing_indicators_enabled,omitempty"`
	LinkPreviewsEnabled          *bool `json:"link_previews_enabled,omitempty"`
	AutoDownloadMedia            *bool `json:"auto_download_media,omitempty"`
}

// ConversationSettingsRepository handles DB access for conversation settings.
type ConversationSettingsRepository interface {
	GetSettings(ctx context.Context, conversationID uuid.UUID) (*ConversationSettings, error)
	UpdateSettings(ctx context.Context, conversationID uuid.UUID, input ConversationSettingsInput) error
}

type conversationSettingsRepository struct {
	db     database.Database
	logger logger.Logger
}

func NewConversationSettingsRepository(db database.Database, log logger.Logger) ConversationSettingsRepository {
	return &conversationSettingsRepository{db: db, logger: log}
}

func (r *conversationSettingsRepository) GetSettings(ctx context.Context, conversationID uuid.UUID) (*ConversationSettings, error) {
	query := `
		SELECT id, conversation_id, disappearing_messages_enabled, disappearing_messages_duration,
		       read_receipts_enabled, typing_indicators_enabled, link_previews_enabled,
		       auto_download_media, created_at, updated_at
		FROM messages.conversation_settings
		WHERE conversation_id = $1
	`

	var s ConversationSettings
	err := r.db.QueryRow(ctx, query, conversationID).Scan(
		&s.ID, &s.ConversationID, &s.DisappearingMessagesEnabled, &s.DisappearingMessagesDuration,
		&s.ReadReceiptsEnabled, &s.TypingIndicatorsEnabled, &s.LinkPreviewsEnabled,
		&s.AutoDownloadMedia, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to get conversation settings",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	return &s, nil
}

func (r *conversationSettingsRepository) UpdateSettings(ctx context.Context, conversationID uuid.UUID, input ConversationSettingsInput) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if input.DisappearingMessagesEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("disappearing_messages_enabled = $%d", argIdx))
		args = append(args, *input.DisappearingMessagesEnabled)
		argIdx++
	}

	if input.DisappearingMessagesDuration != nil {
		setClauses = append(setClauses, fmt.Sprintf("disappearing_messages_duration = $%d", argIdx))
		args = append(args, *input.DisappearingMessagesDuration)
		argIdx++
	}

	if input.ReadReceiptsEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("read_receipts_enabled = $%d", argIdx))
		args = append(args, *input.ReadReceiptsEnabled)
		argIdx++
	}

	if input.TypingIndicatorsEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("typing_indicators_enabled = $%d", argIdx))
		args = append(args, *input.TypingIndicatorsEnabled)
		argIdx++
	}

	if input.LinkPreviewsEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("link_previews_enabled = $%d", argIdx))
		args = append(args, *input.LinkPreviewsEnabled)
		argIdx++
	}

	if input.AutoDownloadMedia != nil {
		setClauses = append(setClauses, fmt.Sprintf("auto_download_media = $%d", argIdx))
		args = append(args, *input.AutoDownloadMedia)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, conversationID)
	query := fmt.Sprintf("UPDATE messages.conversation_settings SET %s WHERE conversation_id = $%d", strings.Join(setClauses, ", "), argIdx)
	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to update conversation settings",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
	}
	return err
}
