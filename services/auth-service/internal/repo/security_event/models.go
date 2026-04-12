package security_event

import "shared/pkg/database/postgres/models"

type SecurityEventInput struct {
	UserID      string
	SessionID   *string
	Status      string
	Description string
	UserAgent   string
	DeviceID    string
}

type enrichedEventInput struct {
	SecurityEventInput
	eventType     models.SecurityEventType
	eventCategory string
	severity      models.SecuritySeverity
}
