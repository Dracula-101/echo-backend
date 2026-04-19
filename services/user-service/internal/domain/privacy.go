package domain

import (
	dbModels "shared/pkg/database/postgres/models"
	"time"
)

type PrivacyOverride struct {
	ID                      string
	UserID                  string
	TargetUserID            string
	LastSeenVisible         *bool
	OnlineStatusVisible     *bool
	ProfilePhotoVisible     *bool
	AboutVisible            *bool
	ReadReceiptsEnabled     *bool
	TypingIndicatorsEnabled *bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewPrivacyOverride(model dbModels.PrivacyOverride) *PrivacyOverride {
	return &PrivacyOverride{
		ID:                      model.ID,
		UserID:                  model.UserID,
		TargetUserID:            model.TargetUserID,
		LastSeenVisible:         model.LastSeenVisible,
		OnlineStatusVisible:     model.OnlineStatusVisible,
		ProfilePhotoVisible:     model.ProfilePhotoVisible,
		AboutVisible:            model.AboutVisible,
		ReadReceiptsEnabled:     model.ReadReceiptsEnabled,
		TypingIndicatorsEnabled: model.TypingIndicatorsEnabled,
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
	}
}
