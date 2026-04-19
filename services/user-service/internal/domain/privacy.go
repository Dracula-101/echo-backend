package domain

import "time"

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
